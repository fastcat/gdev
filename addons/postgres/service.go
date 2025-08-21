package postgres

import (
	"fmt"
	"strconv"

	apiAppsV1 "k8s.io/api/apps/v1"
	apiCoreV1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	applyAppsV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyMetaV1 "k8s.io/client-go/applyconfigurations/meta/v1"

	"fastcat.org/go/gdev/addons/k8s"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
)

func Service(
	opts ...pgSvcOpt,
) service.Service {
	cfg := newSvcConfig(opts...)
	resources := []resource.Resource{
		cfg.pvc(),
		cfg.deployment(),
		cfg.service(),
		cfg.credentialsSecret(),
	}
	if cfg.nodePort > 0 {
		resources = append(resources, cfg.nodePortService())
	}
	if cfg.waitReady {
		resources = append(resources, k8s.DeploymentReadyWaiter(cfg.name))
	}
	if len(cfg.initDBNames) > 0 {
		if cfg.nodePort <= 0 {
			panic(fmt.Errorf("initializing PG DBs requires enabling postgres.WithNodePort"))
		}
		resources = append(resources, cfg.initDBs())
	}
	return service.New(
		cfg.name,
		service.WithResources(resources...),
	)
}

func newSvcConfig(opts ...pgSvcOpt) pgSvcConfig {
	var cfg pgSvcConfig
	for _, o := range opts {
		o(&cfg)
	}
	cfg.fillDefaults()
	return cfg
}

func (c *pgSvcConfig) fillDefaults() {
	if c.major == 0 {
		c.major = DefaultMajor
	}
	if c.variant == nil {
		c.variant = internal.Ptr(DefaultVariant)
	}
	if c.name == "" {
		c.name = fmt.Sprintf("postgres-%d", c.major)
	}
}

type pgSvcConfig struct {
	name        string
	major       int
	variant     *string
	nodePort    int
	initDBNames []string
	waitReady   bool
}

// CredentialsSecretName returns the k8s secret name where credentials will be
// stored (in the form of `PG...` environment variable names) for the default
// service configured in the addon.
func CredentialsSecretName() string {
	addon.CheckInitialized()
	cfg := newSvcConfig(addon.Config.svcOpts...)
	return cfg.CredentialsSecretName()
}

func (c pgSvcConfig) CredentialsSecretName() string {
	if c.name == "" {
		panic(fmt.Errorf("cannot get credentials secret without a service name"))
	}
	return c.name + "-credentials"
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

// Enables or disables exposing the postgres instance on a k8s NodePort.
//
// If port is 0, the NodePort will be disabled. Else the value is the exposed
// port number.
//
// If port is not in [0..65535], it will panic.
func WithNodePort(port int) pgSvcOpt {
	if port < 0 || port > 65535 {
		panic(fmt.Errorf("invalid tcp port %d", port))
	}
	return func(c *pgSvcConfig) {
		c.nodePort = port
	}
}

// WithInitDBs will cause the postgres service to ensure the named databases
// exist during startup.
//
// This requires enabling WithNodePort.
func WithInitDBs(dbs ...string) pgSvcOpt {
	if len(dbs) == 0 {
		panic(fmt.Errorf("must provide at least one init db"))
	}
	return func(c *pgSvcConfig) {
		c.waitReady = true
		c.initDBNames = append(c.initDBNames, dbs...)
	}
}

// WithWaitReady will include a resource in the service that waits for postgres
// to be ready before continuing.
//
// This is implied if you use WithInitDBs.
func WithWaitReady() pgSvcOpt {
	return func(c *pgSvcConfig) {
		c.waitReady = true
	}
}

const DefaultPort = 5432

const pgDataDir = "/var/lib/postgresql/data"

func (c pgSvcConfig) pvc() k8s.Resource {
	pvc := applyCoreV1.PersistentVolumeClaim(c.pvcName(), "").
		// TODO: standard labels & annotations
		WithSpec(applyCoreV1.PersistentVolumeClaimSpec().
			// omit storage class so we get the default one, which should be
			// local-path under k3s
			WithAccessModes(apiCoreV1.ReadWriteOnce).
			// resource limits are not honored, and we couldn't set a good limit if
			// they were, so skip it. at least a request is required however.
			WithResources(applyCoreV1.VolumeResourceRequirements().
				WithRequests(apiCoreV1.ResourceList{
					apiCoreV1.ResourceStorage: apiResource.MustParse("1Gi"),
				}),
			),
		)
	return k8s.PersistentVolumeClaim(pvc)
}

const containerPortName = "postgres"

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
				WithName(containerPortName).
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
					WithPort(DefaultPort).
					WithTargetPort(intstr.FromString(containerPortName)),
			).
			WithSelector(c.selector()),
	)
	return k8s.Service(s)
}

func (c pgSvcConfig) nodePortService() k8s.Resource {
	s := applyCoreV1.Service(c.name+"-node", "").WithSpec(
		applyCoreV1.ServiceSpec().
			// TODO: support changing all these options
			WithType(apiCoreV1.ServiceTypeNodePort).
			WithPorts(
				applyCoreV1.ServicePort().
					WithName("postgresql-node").
					WithAppProtocol("postgresql").
					WithProtocol(apiCoreV1.ProtocolTCP).
					WithPort(DefaultPort).
					WithTargetPort(intstr.FromString(containerPortName)).
					WithNodePort(int32(c.nodePort)),
			).
			WithSelector(c.selector()),
	)
	return k8s.Service(s)
}

func (c pgSvcConfig) credentialsSecret() k8s.Resource {
	s := applyCoreV1.Secret(c.CredentialsSecretName(), "").
		WithStringData(c.Credentials())
	return k8s.Secret(s)
}

// Credentials returns connection credentials (in the form of `PG...`
// environment variable names) for the default service configured in the addon,
// e.g. for setting in a PM service environment block.
func Credentials() map[string]string {
	addon.CheckInitialized()
	cfg := newSvcConfig(addon.Config.svcOpts...)
	return cfg.Credentials()
}

func (c pgSvcConfig) Credentials() map[string]string {
	return map[string]string{
		"PGUSER":     "postgres",
		"PGPASSWORD": internal.AppName(),
	}
}

func (c pgSvcConfig) selector() map[string]string {
	return map[string]string{
		"app": c.name,
	}
}

func (c pgSvcConfig) pvcName() string {
	return c.name + "-data"
}
