package k8s

import (
	"context"
	"fmt"

	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
)

// appliable provides a generic partial implementation of
// [resource.Resource] for most k8s objects that can use the strongly typed
// server side apply pattern.
type appliable[
	Client client[Resource, Apply],
	Resource any,
	Apply apply[Apply],
] struct {
	acc   accessor[Client, Resource, Apply]
	apply Apply
}

func newApply[
	Client client[Resource, Apply],
	Resource any,
	Apply apply[Apply],
](
	acc accessor[Client, Resource, Apply],
	apply Apply,
) appliable[Client, Resource, Apply] {
	m, o := acc.applyMeta(apply)
	if internal.ValueOrZero(m.Kind) == "" {
		panic(fmt.Errorf("require TypeMeta.Name for %T", apply))
	}
	if internal.ValueOrZero(o.Name) == "" {
		panic(fmt.Errorf("require ObjectMeta.Name for %s", *m.Kind))
	}
	// TODO: add standard annotations and labels
	return appliable[Client, Resource, Apply]{acc, apply}
}

// ID implements resource.Resource.
func (r *appliable[Client, Resource, Apply]) ID() string {
	m, o := r.acc.applyMeta(r.apply)
	return "k8s/" + *m.Kind + "/" + *o.Name
}

// Start implements resource.Resource.
func (r *appliable[Client, Resource, Apply]) Start(ctx context.Context) error {
	sc := r.client(ctx)
	// TODO: preserve scale settings if the resource already exists
	if _, err := sc.Apply(ctx, r.apply, applyOpts(ctx)); err != nil {
		m, o := r.acc.applyMeta(r.apply)
		return fmt.Errorf("failed to apply %s %s: %w", *m.Kind, *o.Name, err)
	}
	return nil
}

// Stop implements resource.Resource.
func (r *appliable[Client, Resource, Apply]) Stop(ctx context.Context) error {
	sc := r.client(ctx)
	m, o := r.acc.applyMeta(r.apply)
	if err := sc.Delete(ctx, *o.Name, deleteOpts(ctx)); err != nil && !apiErrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s %s: %w", *m.Kind, *o.Name, err)
	}
	return nil
}

// Ready implements resource.Resource.
func (r *appliable[Client, Resource, Apply]) Ready(ctx context.Context) (bool, error) {
	obj, err := r.client(ctx).Get(ctx, r.K8SName(), getOpts(ctx))
	if err != nil {
		return false, err
	}
	return r.acc.ready(ctx, obj)
}

// K8SKind implements ContainerResource.
func (r *appliable[Client, Resource, Apply]) K8SKind() string {
	m, _ := r.acc.applyMeta(r.apply)
	return *m.Kind
}

// K8SName implements ContainerResource.
func (r *appliable[Client, Resource, Apply]) K8SName() string {
	return *r.apply.GetName()
}

// K8SNamespace implements ContainerResource.
func (r *appliable[Client, Resource, Apply]) K8SNamespace() string {
	_, o := r.acc.applyMeta(r.apply)
	return *o.Namespace
}

func (r *appliable[Client, Resource, Apply]) client(ctx context.Context) Client {
	return r.acc.getClient(
		resource.ContextValue[kubernetes.Interface](ctx),
		resource.ContextValue[Namespace](ctx),
	)
}
