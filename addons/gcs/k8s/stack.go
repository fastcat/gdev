package gcs_k8s

import (
	"context"

	"fastcat.org/go/gdev/addons/gcs/internal"
	"fastcat.org/go/gdev/addons/k8s"
	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
	"fastcat.org/go/gdev/stack"
)

func WithK8SService() internal.Option {
	return func(cfg *internal.Config) {
		cfg.StackHooks = append(cfg.StackHooks, setupK8SService)
	}
}

func setupK8SService(cfg *internal.Config) error {
	stack.AddInfrastructure(service.New(
		"fake-gcs-server",
		service.WithResourceFuncs(func(ctx context.Context) []resource.Resource {
			pvc := k8s.PersistentVolumeClaim(nil) // FIXME
			sr := k8s.Service(nil)                // FIXME
			dr := k8s.Deployment(nil)             // FIXME
			return []resource.Resource{pvc, sr, dr}
		}),
	))
	return nil
}
