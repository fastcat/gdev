package k8s

import (
	"fmt"

	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
)

// applyResource provides a generic partial implementation of
// [resource.Resource] for most k8s objects that can use the strongly typed
// server side apply pattern.
type applyResource[
	Client client[Resource, Apply],
	Resource any,
	Apply any,
] struct {
	acc   accessor[Client, Resource, Apply]
	apply Apply
}

func newApply[
	Client client[Resource, Apply],
	Resource any,
	Apply any,
](
	acc accessor[Client, Resource, Apply],
	apply Apply,
) applyResource[Client, Resource, Apply] {
	m, o := acc.applyMeta(&apply)
	if internal.ValueOrZero(m.Kind) == "" {
		panic(fmt.Errorf("require TypeMeta.Name for %T", apply))
	}
	if internal.ValueOrZero(o.Name) == "" {
		panic(fmt.Errorf("require ObjectMeta.Name for %s", *m.Kind))
	}
	// TODO: add standard annotations and labels
	return applyResource[Client, Resource, Apply]{acc, apply}
}

// ID implements resource.Resource.
func (r *applyResource[Client, Resource, Apply]) ID() string {
	m, o := r.acc.applyMeta(&r.apply)
	return "k8s/" + *m.Kind + "/" + *o.Name
}

// Start implements resource.Resource.
func (r *applyResource[Client, Resource, Apply]) Start(ctx *resource.Context) error {
	sc := r.client(ctx)
	// TODO: preserve scale settings if the resource already exists
	if _, err := sc.Apply(ctx, &r.apply, applyOpts(ctx)); err != nil {
		m, o := r.acc.applyMeta(&r.apply)
		return fmt.Errorf("failed to apply %s %s: %w", *m.Kind, *o.Name, err)
	}
	return nil
}

// Stop implements resource.Resource.
func (r *applyResource[Client, Resource, Apply]) Stop(ctx *resource.Context) error {
	sc := r.client(ctx)
	m, o := r.acc.applyMeta(&r.apply)
	if err := sc.Delete(ctx, *o.Name, deleteOpts(ctx)); err != nil && !apiErrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s %s: %w", *m.Kind, *o.Name, err)
	}
	return nil
}

func (r *applyResource[Client, Resource, Apply]) client(ctx *resource.Context) Client {
	return r.acc.getClient(
		resource.ContextValue[kubernetes.Interface](ctx),
		resource.ContextValue[Namespace](ctx),
	)
}
