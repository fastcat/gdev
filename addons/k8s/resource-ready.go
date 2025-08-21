package k8s

import (
	"context"
	"errors"

	apiErrors "k8s.io/apimachinery/pkg/api/errors"

	"fastcat.org/go/gdev/resource"
)

// APIReadyWaiter constructs a resource that will block waiting for the k8s api
// server to be ready during startup.
//
// Kubernetes providers such as k3s can include this in their startup resource
// so that other services that want to talk to k8s don't hit errors from
// starting too early.
func APIReadyWaiter() resource.Resource {
	return resource.Waiter("k8s-api-ready", func(ctx context.Context) (bool, error) {
		client := resource.ContextValue[Interface](ctx)
		if err := client.Health().Ready(ctx); err != nil {
			var se *apiErrors.StatusError
			if errors.As(err, &se) {
				switch se.ErrStatus.Code {
				case 503:
					// not ready yet
					return false, nil
				case 500:
					// some components aren't ready yet, message will generally list which
					// ones, but we don't care until we have logging going
					return false, nil
				default:
					return false, err
				}
			}
			return false, err
		}
		return true, nil
	})
}

// NodeReadyWaiter creates a Resource that will block waiting for at least one
// Node to be registered, and for all the registered nodes to be ready.
//
// Kubernetes providers such as k3s can include this in their startup so that
// other resources don't try to start when k8s has nowhere to run things.
func NodeReadyWaiter() resource.Resource {
	return resource.Waiter("k8s-node-ready", func(ctx context.Context) (bool, error) {
		client := resource.ContextValue[Interface](ctx)
		l, err := accNode.list(ctx, client.CoreV1().Nodes(), listOpts(ctx))
		if err != nil {
			return false, err
		}
		for i := range l {
			if ready, err := accNode.ready(ctx, &l[i]); err != nil {
				return false, err
			} else if !ready {
				return false, nil
			}
		}
		return len(l) > 0, nil
	})
}
