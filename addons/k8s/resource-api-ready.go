package k8s

import (
	"context"
	"errors"

	apiErrors "k8s.io/apimachinery/pkg/api/errors"

	"fastcat.org/go/gdev/resource"
)

// APIReady constructs a resource that will block waiting for the k8s api server
// to be ready during startup.
//
// Kubernetes providers such as k3s can include this in their startup resource
// so that other services that want to talk to k8s don't hit errors from
// starting too early.
func APIReadyWaiter() resource.Resource {
	return resource.NewWaitResource(
		"k8s-api-ready",
		func(ctx context.Context) (bool, error) {
			client := resource.ContextValue[Interface](ctx)
			if err := client.Health().Ready(ctx); err != nil {
				var se *apiErrors.StatusError
				if errors.As(err, &se) {
					switch se.ErrStatus.Code {
					case 503: // Service Unavailable
						// not ready yet
						return false, nil
					}
				}
				return false, err
			}
			return true, nil
		},
	)
}
