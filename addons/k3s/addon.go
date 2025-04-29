package k3s

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/k8s"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/pm/api"
	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
	"fastcat.org/go/gdev/stack"
	"fastcat.org/go/gdev/sys"
	apiCoreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
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

	k8s.Configure(
		k8s.WithContextFunc(addon.Config.ContextName),
		k8s.WithNamespace(string(addon.Config.namespace)),
	)

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

func initialize() error {
	bootstrap.AddStep(bootstrap.Step("Install k3s",
		func(ctx *bootstrap.Context) error {
			return InstallStable(ctx, DefaultInstallPath)
		},
		// TODO: sim invoker that will still read the release data
	))

	// TODO: this isn't in the right place, as the k3s kube config won't exist to
	// merge from until after k3s is running.
	resource.AddContextEntry(mergeKubeConfig)

	// TODO: resource context setup

	// TODO: hook into a bootstrap system to install/uninstall k3s

	// TODO: hook into stop to add a way to kill off all the k3s containers. this
	// is easy with docker but harder with containerd once k3s (which _is_
	// containerd) is gone.

	addStackService(&addon.Config)

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

func addStackService(cfg *config) {
	stack.AddService(service.NewService(
		"k3s",
		service.WithResources(
			// TODO: add a stop-only resource that stops the systemd user unit it ran
			// under to get rid of all the pods, and then uses systemd apis(?) to find
			// the containerd-shim-... processes to kill as well. Goes here because
			// resources are stopped in reverse order, so it should run after k3s
			// itself is stopped.
			resource.PMStatic(api.Child{
				// TODO: flag this service to not be restarted on stack "apply"
				Name: "k3s",
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
				Init: []api.Exec{{
					// try to kill running k3s before trying to start a new one
					Cwd:  "/",
					Cmd:  "/bin/sh",
					Args: []string{"-c", "sudo -n pkill -TERM k3s || true"},
				}},
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
	))
}

type clientConfigMarker struct{}

// merge k3s config with user config under the configured name
func mergeKubeConfig(ctx context.Context) (clientConfigMarker, error) {
	addon.CheckInitialized()
	var ret clientConfigMarker
	const k3sFn = "/etc/rancher/k3s/k3s.yaml"
	k3sCfg, err := clientcmd.LoadFromFile(k3sFn)
	if err != nil {
		// often not readable to us, try again with sudo if so
		if !errors.Is(err, os.ErrPermission) {
			return ret, err
		}
		if r, err2 := sys.SudoReader(ctx, k3sFn, false); err2 != nil {
			return ret, fmt.Errorf("failed to read k3s config %s: %w", k3sFn, errors.Join(err, err2))
		} else if content, err := io.ReadAll(r); err != nil {
			if err2 := r.Close(); err2 != nil {
				err = errors.Join(err, err2)
			}
			return ret, fmt.Errorf("failed to read k3s config %s: %w", k3sFn, err)
		} else if err := r.Close(); err != nil {
			return ret, err
		} else if k3sCfg, err = clientcmd.Load(content); err != nil {
			return ret, err
		}
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
