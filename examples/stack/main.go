package main

import (
	"path/filepath"

	"fastcat.org/go/gdev/addons/build"
	"fastcat.org/go/gdev/addons/golang"
	"fastcat.org/go/gdev/addons/pm"
	"fastcat.org/go/gdev/addons/pm/api"
	"fastcat.org/go/gdev/addons/pm/resource"
	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/service"
	"fastcat.org/go/gdev/shx"
	"fastcat.org/go/gdev/stack"
)

// When you have many services using a common pattern, defining a recipe
// function like this is helpful to avoid stuttering of the service name and
// repetition of the resource pattern.
func myStandardService(
	name string,
	cmd string,
	args []string,
	opts ...service.BasicOpt,
) service.Service {
	allOpts := []service.BasicOpt{
		service.WithModalResources(
			service.ModeDefault,
			// this would normally be a container service, but this example doesn't pull in k3s or docker
			resource.PMStatic(api.Child{
				Name: name,
				Main: api.Exec{
					Cmd:  cmd,
					Args: []string{"1h"},
				},
			}),
		),
		service.WithModalResources(
			service.ModeLocal,
			resource.PMStatic(api.Child{
				Name: name + "-local",
				Main: api.Exec{
					Cmd:  cmd,
					Args: args,
				},
			}),
		),
	}
	allOpts = append(allOpts, opts...)
	return service.New(name, allOpts...)
}

func main() {
	// cspell:ignore sdev
	instance.SetAppName("sdev")
	pm.Configure()
	build.Configure()
	golang.Configure()

	svc1Repo := filepath.Join(shx.HomeDir(), "src", "gdev")
	svc1Subdir := "examples/stack"
	stack.AddService(
		myStandardService(
			"svc1",
			"sleep",
			[]string{"1h"},
			service.WithSource(svc1Repo, svc1Subdir, "", ""),
		),
	)
	stack.AddService(service.New("svc2",
		service.WithResources(
			resource.PMStatic(api.Child{
				Name: "svc2",
				Main: api.Exec{
					Cmd:  "sleep",
					Args: []string{"1h"},
				},
			}),
		),
	))

	cmd.Main()
}
