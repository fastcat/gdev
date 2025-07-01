package gcs_docker

import (
	"context"
	"strconv"

	"fastcat.org/go/gdev/addons/docker"
	"fastcat.org/go/gdev/addons/gcs/internal"
	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
	"fastcat.org/go/gdev/stack"
)

func WithDockerService() internal.Option {
	return func(cfg *internal.Config) {
		cfg.StackHooks = append(cfg.StackHooks, setupDockerService)
	}
}

func setupDockerService(cfg *internal.Config) error {
	stack.AddInfrastructure(service.New(
		"fake-gcs-server",
		service.WithResourceFuncs(func(ctx context.Context) []resource.Resource {
			dv := docker.Volume("gcs-data")
			dc := docker.Container("fake-gcs-server", cfg.FakeServerImage).
				WithPorts(strconv.Itoa(cfg.ExposedPort)).
				WithCmd(cfg.Args()...).
				// NOTE: container accepts both a `/data` dir (for preload) and a
				// `/storage` dir where it saves the bucket data.
				WithVolumeMount(dv.Name, "/storage")
			return []resource.Resource{
				dv,
				dc,
			}
		}),
	))
	return nil
}
