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
			// dv := docker.Volume() // FIXME
			dc := docker.Container(
				"fake-gcs-server",
				cfg.FakeServerImage,
				[]string{strconv.Itoa(cfg.ExposedPort)}, // FIXME?
				map[string]string{
					// FIXME
				},
			)
			return []resource.Resource{
				dc,
			}
		}),
	))
	return nil
}
