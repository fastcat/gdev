package main

import (
	"path/filepath"

	"fastcat.org/go/gdev/addons/build"
	"fastcat.org/go/gdev/addons/docker"
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
	repo, subDir string,
	imageName string,
	opts ...service.BasicOpt,
) service.Service {
	allOpts := []service.BasicOpt{
		service.WithModalResources(
			service.ModeDefault,
			docker.Container(
				name,
				imageName,
				[]string{"8080"},
				nil,
			),
		),
		service.WithModalResources(
			service.ModeLocal,
			resource.PMStatic(api.Child{
				Name: name + "-local",
				Main: api.Exec{
					Cmd:  "go",
					Args: []string{"run", filepath.Join(".", subDir)},
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
	docker.Configure()
	build.Configure()
	golang.Configure()

	svc1Repo := filepath.Join(shx.HomeDir(), "src", "gdev")
	svc1Subdir := "examples/stack/svc1"
	stack.AddService(
		myStandardService(
			"svc1",
			svc1Repo, svc1Subdir,
			"ghcr.io/fastcat/gdev/svc1",
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
