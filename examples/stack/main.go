package main

import (
	"fastcat.org/go/gdev/addons/pm"
	"fastcat.org/go/gdev/addons/pm/api"
	"fastcat.org/go/gdev/addons/pm/resource"
	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/service"
	"fastcat.org/go/gdev/stack"
)

func main() {
	// cspell:ignore sdev
	instance.SetAppName("sdev")
	pm.Configure()

	// TODO: lots of stuttering here
	stack.AddService(service.New("svc1",
		service.WithResources(
			resource.PMStatic(api.Child{
				Name: "svc1",
				Main: api.Exec{
					Cmd:  "sleep",
					Args: []string{"1h"},
				},
			}),
		),
	))
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
