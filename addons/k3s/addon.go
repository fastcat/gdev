package k3s

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/spf13/cobra"
	apiCoreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/k8s"
	"fastcat.org/go/gdev/addons/pm"
	"fastcat.org/go/gdev/addons/pm/api"
	pmResource "fastcat.org/go/gdev/addons/pm/resource"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
	"fastcat.org/go/gdev/stack"
	"fastcat.org/go/gdev/sys"
)

var addon = addons.Addon[config]{
	Config: config{
		// contextName defaults to a late bind based on the app name
		namespace: k8s.Namespace(apiCoreV1.NamespaceDefault),
		k3sPath:   DefaultInstallPath,
	},
}

type provider struct {
	desc      string
	configure func()
}

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}
	if addon.Config.provider == nil {
		panic(errors.New("must select a k3s container provider (containerd or docker)"))
	}

	configurePM()
	// don't once this, overwrite the settings if they change
	k8s.Configure(
		k8s.WithContextFunc(addon.Config.ContextName),
		k8s.WithNamespace(string(addon.Config.namespace)),
	)
	configureBootstrap() // once
	addon.Config.provider.configure()

	addon.RegisterIfNeeded(addons.Definition{
		Name: "k3s",
		Description: func() string {
			internal.CheckLockedDown()
			return "Support running k3s for local kubernetes, using " +
				addon.Config.provider.desc +
				", context " + addon.Config.ContextName() +
				", and namespace " + string(addon.Config.namespace)
		},
		Initialize: initialize,
	})
}

var configurePM = sync.OnceFunc(func() {
	pm.Configure()
})

var configureBootstrap = sync.OnceFunc(func() {
	k3sCmd := &cobra.Command{
		Use: "k3s",
	}
	k3sCmd.AddCommand(
		&cobra.Command{
			Use:   "install",
			Short: "install / update k3s",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				if err := InstallStable(cmd.Context(), DefaultInstallPath); err != nil {
					return err
				}
				if err := InstallSudoers(cmd.Context(), DefaultInstallPath); err != nil {
					return err
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "setup",
			Short: "do first time start to setup baseline k3s configuration",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				ctx := resource.NewEmptyContext(cmd.Context())
				rs := stackService(&addon.Config).Resources(ctx)
				for _, r := range rs {
					if err := r.Start(ctx); err != nil {
						return err
					}
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "cleanup-containerd",
			Short: "kill any containerd pods left running",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				return killPods(cmd.Context())
			},
		},
		&cobra.Command{
			Use:   "uninstall",
			Short: "uninstall k3s",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				return Uninstall(cmd.Context(), DefaultInstallPath)
			},
		},
	)

	bootstrap.Configure(
		bootstrap.WithSteps(bootstrap.Step("Install k3s",
			func(ctx *bootstrap.Context) error {
				return InstallStable(ctx, DefaultInstallPath)
			},
			// TODO: sim invoker that will still read the release data
		)),
		bootstrap.WithSteps(bootstrap.Step("Install sudoers to run k3s",
			func(ctx *bootstrap.Context) error {
				return InstallSudoers(ctx, DefaultInstallPath)
			},
			// TODO: sim invoker that will still read the release data
		)),
		bootstrap.WithChildCmds(k3sCmd),
	)
})

func initialize() error {
	// TODO: this isn't in the right place, as the k3s kube config won't exist to
	// merge from until after k3s is running.
	resource.AddContextEntry(mergeKubeConfig)

	// TODO: resource context setup

	// TODO: hook into a bootstrap system to install/uninstall k3s

	// TODO: hook into stop to add a way to kill off all the k3s containers. this
	// is easy with docker but harder with containerd once k3s (which _is_
	// containerd) is gone.

	stack.AddInfrastructure(stackService(&addon.Config))
	// TODO: add infra service to wait for kube to be ready to run pods in the
	// selected namespace: it exists, and at least one node is ready. except this
	// really belongs in the k8s addon, but that produces an ordering issue.

	addon.Initialized()

	return nil
}

// TODO: this addon's config is mostly a copy of the k8s addon config

type config struct {
	contextName string
	namespace   k8s.Namespace
	provider    *provider
	k3sPath     string
	k3sArgs     []string
}

type option func(*config)

func WithContext(name string) option {
	return func(ac *config) {
		ac.contextName = name
	}
}

func WithNamespace(name string) option {
	return func(ac *config) {
		ac.namespace = k8s.Namespace(name)
	}
}

// WithPath sets the absolute path to the k3s binary
func WithPath(k3sPath string) option {
	if !filepath.IsAbs(k3sPath) {
		panic(fmt.Errorf("k3s path %q is not absolute", k3sPath))
	}
	return func(ac *config) {
		ac.k3sPath = k3sPath
	}
}

// WithK3SArgs adds extra CLI args to the k3s invocation
func WithK3SArgs(args ...string) option {
	return func(ac *config) {
		ac.k3sArgs = append(ac.k3sArgs, args...)
	}
}

func WithProvider(
	desc string,
	k3sArgs []string,
	configure func(),
) option {
	return func(ac *config) {
		if ac.provider != nil {
			panic(errors.New("already have a provider"))
		}
		ac.provider = &provider{
			desc:      desc,
			configure: configure,
		}
		ac.k3sArgs = append(ac.k3sArgs, k3sArgs...)
	}
}

func (c *config) ContextName() string {
	internal.CheckLockedDown()
	if c.contextName != "" {
		return c.contextName
	}
	return instance.AppName()
}

func stackService(cfg *config) service.Service {
	return service.NewService(
		"k3s",
		service.WithResources(
			// TODO: add a stop-only resource that stops the systemd user unit it ran
			// under to get rid of all the pods, and then uses systemd apis(?) to find
			// the containerd-shim-... processes to kill as well. Goes here because
			// resources are stopped in reverse order, so it should run after k3s
			// itself is stopped.
			pmResource.PMStaticInfra(api.Child{
				// TODO: flag this service to not be restarted on stack "apply"
				Name: "k3s",
				Annotations: map[string]string{
					// TODO: we want this annotation to be automatic due to it being part
					// of the infrastructure service list.
					api.AnnotationGroup: "infrastructure",
				},
				Init: []api.Exec{{
					// try to kill running k3s before trying to start a new one
					Cwd:  "/",
					Cmd:  "/bin/sh",
					Args: []string{"-c", "sudo -n pkill -TERM k3s || true"},
				}},
				Main: api.Exec{
					Cwd: "/", // TODO: $HOME?
					// TODO: support running k3s not as root
					Cmd: "sudo",
					Args: append(
						[]string{
							"-n",
							cfg.k3sPath,
							"server",
						},
						cfg.k3sArgs...,
					),
				},
				HealthCheck: &api.HealthCheck{
					TimeoutSeconds: 1,
					Http: &api.HttpHealthCheck{
						Scheme: "https",
						// TODO: provide the certs to validate this somehow
						Insecure: true,
						Port:     6443,
						Path:     "/ping",
					},
				},
			}),
		),
		// TODO: add a "waiter" resource for k3s to be ready: not just pinging, but
		// the local node healthy too.
	)
}

type clientConfigMarker struct{}

// merge k3s config with user config under the configured name
func mergeKubeConfig(ctx context.Context) (clientConfigMarker, error) {
	addon.CheckInitialized()
	var ret clientConfigMarker
	const k3sFn = "/etc/rancher/k3s/k3s.yaml"
	content, err := sys.ReadFileAsRoot(ctx, k3sFn, false)
	if err != nil {
		return ret, fmt.Errorf("failed to read k3s kube config: %w", err)
	}
	k3sCfg, err := clientcmd.Load(content)
	if err != nil {
		return ret, fmt.Errorf("failed to parse k3s config: %w", err)
	}

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	userFn := rules.GetDefaultFilename()
	userCfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, nil).RawConfig()
	if err != nil {
		return ret, fmt.Errorf("failed loading user kube config %s: %w", userFn, err)
	}

	// copy the settings into the user's config, with some edits, if needed
	dirty := false
	name := addon.Config.ContextName()
	if !reflect.DeepEqual(userCfg.Clusters[name], k3sCfg.Clusters["default"]) {
		dirty = true
		userCfg.Clusters[name] = k3sCfg.Clusters["default"]
	}
	if !reflect.DeepEqual(userCfg.AuthInfos[name], k3sCfg.AuthInfos["default"]) {
		dirty = true
		userCfg.AuthInfos[name] = k3sCfg.AuthInfos["default"]
	}
	context := k3sCfg.Contexts["default"]
	context.AuthInfo = name
	context.Cluster = name
	if !reflect.DeepEqual(userCfg.Contexts[name], context) {
		dirty = true
		userCfg.Contexts[name] = context
	}

	if !dirty {
		return ret, nil
	}

	// write the user config back out
	if err := clientcmd.WriteToFile(userCfg, userFn); err != nil {
		return ret, fmt.Errorf("failed to write user kube config %s: %w", userFn, err)
	}

	return ret, nil
}
