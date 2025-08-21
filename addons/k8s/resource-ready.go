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
			if ready, err := accNode.ready(ctx, client, &l[i]); err != nil {
				return false, err
			} else if !ready {
				return false, nil
			}
		}
		return len(l) > 0, nil
	})
}

// DeploymentReadyWaiter creates a Resource that will block waiting for the
// named Deployment to be ready. It will error out if the deployment does not
// exist.
func DeploymentReadyWaiter(name string) resource.Resource {
	return accReadyWaiter(accDeployment, name)
}

// StatefulsetReadyWaiter creates a Resource that will block waiting for the
// named StatefulSet to be ready. It will error out if the StatefulSet does not
// exist.
func StatefulsetReadyWaiter(name string) resource.Resource {
	return accReadyWaiter(accStatefulSet, name)
}

// ServiceReadyWaiter creates a Resource that will block waiting for the
// named Service to be ready. It will error out if the Service does not
// exist. Ready for a service means at least one healthy endpoint.
func ServiceReadyWaiter(name string) resource.Resource {
	return accReadyWaiter(accService, name)
}

func accReadyWaiter[
	Client client[Resource, Apply],
	Resource any,
	Apply apply[Apply],
](acc accessor[Client, Resource, Apply], name string) resource.Resource {
	return resource.Waiter(acc.typ.Kind+"/"+name, func(ctx context.Context) (bool, error) {
		kc := resource.ContextValue[Interface](ctx)
		namespace := resource.ContextValue[Namespace](ctx)
		c := acc.getClient(kc, namespace)
		r, err := c.Get(ctx, name, getOpts(ctx))
		if err != nil {
			return false, err
		}
		return acc.ready(ctx, kc, r)
	})
}
