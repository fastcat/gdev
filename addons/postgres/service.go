package postgres

import (
	"fmt"
	"strconv"

	apiAppsV1 "k8s.io/api/apps/v1"
	apiCoreV1 "k8s.io/api/core/v1"
	applyAppsV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyMetaV1 "k8s.io/client-go/applyconfigurations/meta/v1"

	"fastcat.org/go/gdev/addons/k8s"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/service"
)

func Service(
	opts ...pgSvcOpt,
) service.Service {
	var cfg pgSvcConfig
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.major == 0 {
		cfg.major = DefaultMajor
	}
	if cfg.variant == nil {
		cfg.variant = internal.Ptr(DefaultVariant)
	}
	if cfg.name == "" {
		cfg.name = fmt.Sprintf("postgres-%d", cfg.major)
	}
	return service.NewService(
		cfg.name,
		service.WithResources(
			cfg.pvc(),
			cfg.deployment(),
			cfg.service(),
		),
	)
}

type pgSvcConfig struct {
	name    string
	major   int
	variant *string
}

type pgSvcOpt func(c *pgSvcConfig)

// Set the service name, determines the stack service name, k8s deployment name,
// and k8s service name.
//
// If name is not set (or set to the empty string), a
// default name will be chosen based on the major version.
func WithName(name string) pgSvcOpt {
	return func(c *pgSvcConfig) {
		c.name = name
	}
}

const DefaultMajor = 17

// Set the major version of postgres to run. Different major versions will have
// data stored in different PVs.
//
// If this is not set, or set to zero, [DefaultMajor] will be used.
func WithMajor(major int) pgSvcOpt {
	if major < 0 || major > 0 && major < 10 {
		panic(fmt.Errorf("invalid postgres major version %d", major))
	}
	return func(c *pgSvcConfig) {
		c.major = major
	}
}

// The default variant of the image to use.
//
// Note that this is may not be the same as the upstream default variant!
const DefaultVariant = "alpine"

// Set the variant to run. Changing variants between Debian and Alpine will
// cause data to need to be re-indexed in most cases due to differeing libc
// implementations. Different variants will NOT have their data stored in
// different PVs.
//
// If unset [DefaultVariant] will be used.
// If set to the empty string, the default variant at the image level will be used.
func WithVariant(variant string) pgSvcOpt {
	// TODO: validate valid variants
	return func(c *pgSvcConfig) {
		c.variant = &variant
	}
}

const DefaultPort = 5432

const pgDataDir = "/var/lib/postgresql/data"

func (c pgSvcConfig) pvc() k8s.Resource {
	pvc := applyCoreV1.PersistentVolumeClaim(c.pvcName(), "").
		// TODO: standard labels & annotations
		WithSpec(applyCoreV1.PersistentVolumeClaimSpec().
			// TODO: this assumes k3s, doesn't allow other providers, nor using longhorn
			// or whatever
			WithStorageClassName("local-path").
			WithAccessModes(apiCoreV1.ReadWriteOnce),
		// resource requests (size limit) are not honored, so don't bother setting
		// them here, we'd be hard presseed to decide what it should be anyways.
		)
	return k8s.PersistentVolumeClaim(pvc)
}

func (c pgSvcConfig) deployment() k8s.ContainerResource {
	img := "postgres:" + strconv.Itoa(c.major)
	if internal.ValueOrZero(c.variant) != "" {
		img += "-" + *c.variant
	}
	const volName = "pg-data"
	ready := applyCoreV1.ExecAction().WithCommand("pg_isready", "-U", "postgres")
	startupProbe := applyCoreV1.Probe().
		WithExec(ready).
		WithInitialDelaySeconds(1).
		WithSuccessThreshold(1).
		WithFailureThreshold(300). // recovery can be slow
		WithPeriodSeconds(1).
		WithTimeoutSeconds(1)
	// ready mostly the same as startup
	readyProbe := internal.Ptr(*startupProbe).
		WithFailureThreshold(5).
		WithPeriodSeconds(15).
		WithTimeoutSeconds(15)
	pc := applyCoreV1.Container().
		WithName("postgres").
		WithImage(img).
		// these are floating images, move forward automatically to get bug fixes
		WithImagePullPolicy(apiCoreV1.PullAlways).
		// TODO: allow setting config options, pass as args
		WithPorts(
			applyCoreV1.ContainerPort().
				WithName("postgres").
				WithProtocol(apiCoreV1.ProtocolTCP).
				WithContainerPort(DefaultPort),
		).
		WithEnv(k8s.EnvApply(map[string]string{
			// TODO: allow more customization
			"POSTGRES_PASSWORD": internal.AppName(),
		})...).
		WithStartupProbe(startupProbe).
		WithReadinessProbe(readyProbe).
		WithVolumeMounts(
			applyCoreV1.VolumeMount().
				WithName(volName).
				WithMountPath(pgDataDir),
		)
	ps := applyCoreV1.PodSpec().
		WithVolumes(
			applyCoreV1.Volume().
				WithName(volName).
				WithPersistentVolumeClaim(
					applyCoreV1.PersistentVolumeClaimVolumeSource().WithClaimName(c.pvcName()),
				),
		).
		WithContainers(pc)
	pt := applyCoreV1.PodTemplateSpec().
		WithSpec(ps).
		// TODO: add annotations
		// TODO: add standard labels
		WithLabels(c.selector())
	d := applyAppsV1.Deployment(c.name, "").WithSpec(
		applyAppsV1.DeploymentSpec().
			WithReplicas(1).
			WithSelector(
				applyMetaV1.LabelSelector().WithMatchLabels(c.selector()),
			).
			WithStrategy(
				applyAppsV1.DeploymentStrategy().WithType(apiAppsV1.RecreateDeploymentStrategyType),
			).
			WithTemplate(pt),
	)
	return k8s.Deployment(d)
}

func (c pgSvcConfig) service() k8s.Resource {
	s := applyCoreV1.Service(c.name, "").WithSpec(
		applyCoreV1.ServiceSpec().
			// TODO: support changing all these options
			WithType(apiCoreV1.ServiceTypeClusterIP).
			WithPorts(
				applyCoreV1.ServicePort().
					WithName("postgresql").
					WithAppProtocol("postgresql").
					WithProtocol(apiCoreV1.ProtocolTCP).
					WithPort(DefaultPort),
			).
			WithSelector(c.selector()),
	)
	return k8s.Service(s)
}
func (c pgSvcConfig) selector() map[string]string {
	return map[string]string{
		"app": c.name,
	}
}
func (c pgSvcConfig) pvcName() string {
	return c.name + "-data"
}
