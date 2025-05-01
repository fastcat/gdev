package valkey

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	apiAppsV1 "k8s.io/api/apps/v1"
	apiCoreV1 "k8s.io/api/core/v1"
	applyAppsV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyMetaV1 "k8s.io/client-go/applyconfigurations/meta/v1"

	"fastcat.org/go/gdev/addons/k8s"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/service"
)

// Service returns a service object to run Valkey, a Redis replacement.
//
// The default, absent options overriding, is to run a service named `valkey`
// running the latest release version using [DefaultVariant].
func Service(
	opts ...valkeySvcOpt,
) service.Service {
	var cfg valkeySvcConfig
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.variant == nil {
		cfg.variant = internal.Ptr(DefaultVariant)
	}
	if cfg.name == "" {
		if cfg.major == 0 {
			cfg.name = "valkey"
		} else {
			cfg.name = fmt.Sprintf("valkey-%d", cfg.major)
		}
	}

	return service.NewService(
		cfg.name,
		service.WithResources(
			cfg.configmap(),
			cfg.deployment(),
			cfg.service(),
		),
	)
}

// The default image variant that will be used. Note that this may not be the
// same as upstream's default variant.
const DefaultVariant = "alpine"

type valkeySvcConfig struct {
	name     string
	major    int
	variant  *string
	cfgLines []string
}

type valkeySvcOpt func(c *valkeySvcConfig)

// Set the service name, determines the stack service name, k8s deployment name,
// and k8s service name.
//
// If name is not set (or set to the empty string), a
// default name will be chosen based on the major version.
func WithName(name string) valkeySvcOpt {
	return func(c *valkeySvcConfig) {
		c.name = name
	}
}

// Set the major version of valkey to run.
//
// If this is not set, or set to zero, it will use the latest major version.
func WithMajor(major int) valkeySvcOpt {
	if major < 0 || major > 0 && major < 7 {
		panic(fmt.Errorf("invalid valkey major version %d", major))
	}
	return func(c *valkeySvcConfig) {
		c.major = major
	}
}

// Set the variant to run.
//
// If unset [DefaultVariant] will be used.
// If set to the empty string, the default variant at the image level will be used.
func WithVariant(variant string) valkeySvcOpt {
	// TODO: validate valid variants
	return func(c *valkeySvcConfig) {
		c.variant = &variant
	}
}

// WithConfig adds additional lines to the valkey.conf file.
func WithConfig(lines ...string) valkeySvcOpt {
	return func(c *valkeySvcConfig) {
		c.cfgLines = append(c.cfgLines, lines...)
	}
}

// TODO: persistence support with a PVC

func (c valkeySvcConfig) tag() string {
	if c.major == 0 {
		if c.variant == nil {
			return DefaultVariant
		} else if *c.variant == "" {
			return "latest"
		}
		return *c.variant
	} else if c.variant == nil {
		return strconv.Itoa(c.major) + "-" + DefaultVariant
	} else if *c.variant == "" {
		return strconv.Itoa(c.major)
	} else {
		return strconv.Itoa(c.major) + "-" + *c.variant
	}
}

func (c valkeySvcConfig) configHash() string {
	// this is just a change detection mechanism, don't need a super secure hash
	h := sha1.New()
	for _, l := range c.cfgLines {
		if _, err := h.Write([]byte(l)); err != nil {
			// this should never happen
			panic(err)
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (c valkeySvcConfig) configmap() k8s.Resource {
	confLines := strings.Join(c.cfgLines, "\n")
	cm := applyCoreV1.ConfigMap(c.name, "").
		WithData(map[string]string{
			"valkey.conf": confLines,
		})
	return k8s.ConfigMap(cm)
}

const DefaultPort = 6379 // same as redis

func (c valkeySvcConfig) deployment() k8s.ContainerResource {
	img := "valkey/valkey:" + c.tag()
	ready := applyCoreV1.ExecAction().WithCommand("valkey-cli", "ping")
	startupProbe := applyCoreV1.Probe().
		WithExec(ready).
		WithInitialDelaySeconds(1).
		WithSuccessThreshold(1).
		WithFailureThreshold(15).
		WithPeriodSeconds(1).
		WithTimeoutSeconds(1)
	// ready mostly the same as startup
	readyProbe := internal.Ptr(*startupProbe).
		WithFailureThreshold(5).
		WithPeriodSeconds(15).
		WithTimeoutSeconds(15)
	pc := applyCoreV1.Container().
		WithName("valkey").
		WithImage(img).
		// these are floating images, move forward automatically to get bug fixes
		WithImagePullPolicy(apiCoreV1.PullAlways).
		WithArgs("/conf/valkey.conf").
		// TODO: allow setting config options, pass as args
		WithPorts(
			applyCoreV1.ContainerPort().
				WithName("valkey").
				WithProtocol(apiCoreV1.ProtocolTCP).
				WithContainerPort(DefaultPort),
		).
		WithEnv(k8s.EnvApply(map[string]string{
			// TODO: allow customization?
		})...).
		WithStartupProbe(startupProbe).
		WithReadinessProbe(readyProbe).
		WithVolumeMounts(
			applyCoreV1.VolumeMount().
				WithName("config").
				WithMountPath("/conf"),
		)
	ps := applyCoreV1.PodSpec().
		WithContainers(pc).
		WithVolumes(
			applyCoreV1.Volume().
				WithName("config").
				WithConfigMap(
					applyCoreV1.ConfigMapVolumeSource().
						WithName("valkey").
						WithDefaultMode(0o444),
				),
		)
	pt := applyCoreV1.PodTemplateSpec().
		WithSpec(ps).
		WithAnnotations(map[string]string{
			"config-hash": c.configHash(),
		}).
		// TODO: add standard annotations
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

func (c valkeySvcConfig) service() k8s.Resource {
	s := applyCoreV1.Service(c.name, "").WithSpec(
		applyCoreV1.ServiceSpec().
			// TODO: support changing all these options
			WithType(apiCoreV1.ServiceTypeClusterIP).
			WithPorts(
				applyCoreV1.ServicePort().
					WithName("valkey").
					WithAppProtocol("redis").
					WithProtocol(apiCoreV1.ProtocolTCP).
					WithPort(DefaultPort),
			).
			WithSelector(c.selector()),
	)
	return k8s.Service(s)
}

func (c valkeySvcConfig) selector() map[string]string {
	return map[string]string{
		"app": c.name,
	}
}
