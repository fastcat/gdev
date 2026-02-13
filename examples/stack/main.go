package main

import (
	"path/filepath"

	"fastcat.org/go/gdev/addons/build"
	"fastcat.org/go/gdev/addons/docker"
	"fastcat.org/go/gdev/addons/golang"
	"fastcat.org/go/gdev/addons/nodejs"
	"fastcat.org/go/gdev/addons/pm"
	"fastcat.org/go/gdev/addons/pm/api"
	"fastcat.org/go/gdev/addons/pm/resource"
	"fastcat.org/go/gdev/addons/stack"
	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/lib/shx"
	"fastcat.org/go/gdev/service"
)

// When you have many services using a common pattern, defining a recipe
// function like this is helpful to avoid stuttering of the service name and
// repetition of the resource pattern.
func myGoService(
	name string,
	_repo, subDir string,
	imageName string,
	opts ...service.BasicOpt,
) service.Service {
	allOpts := []service.BasicOpt{
		service.WithModalResources(
			service.ModeDefault,
			docker.Container(name, imageName).WithPorts("8080"),
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

func myNodeService(
	name string,
	_repo, _subDir string,
	imageName string,
	opts ...service.BasicOpt,
) service.Service {
	allOpts := []service.BasicOpt{
		service.WithModalResources(
			service.ModeDefault,
			docker.Container(name, imageName).WithPorts("8081"),
		),
		service.WithModalResources(
			service.ModeLocal,
			resource.PMStatic(api.Child{
				Name: name + "-local",
				Main: api.Exec{
					Cmd:  "node",
					Args: []string{"."},
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
	nodejs.Configure()

	svc1Repo := filepath.Join(shx.HomeDir(), "src", "gdev")
	svc1Subdir := "examples/stack/svc1"
	stack.AddService(
		myGoService(
			"svc1",
			svc1Repo, svc1Subdir,
			"ghcr.io/fastcat/gdev/svc1",
			service.WithSource(svc1Repo, svc1Subdir, "", ""),
		),
	)
	// use this so build detection sees the rush project
	svc2Repo := filepath.Join(shx.HomeDir(), "src", "gdev", "examples", "stack", "svc2")
	svc2Subdir := "app"
	stack.AddService(
		myNodeService(
			"svc2",
			svc2Repo, svc2Subdir,
			"ghcr.io/fastcat/gdev/svc2",
			service.WithSource(svc2Repo, svc2Subdir, "", ""),
		),
	)

	cmd.Main()
}
