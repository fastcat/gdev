package k8s

import (
	"fmt"

	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
	applyAppsV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	"k8s.io/client-go/kubernetes"
)

// podder generalizes the pattern of a k8s resource that schedules pods
type podder[
	Client client[Resource, Apply],
	Resource any,
	Apply any,
] struct {
	acc   accessor[Client, Resource, Apply]
	apply Apply
}

func newPodder[
	Client client[Resource, Apply],
	Resource any,
	Apply any,
](
	acc accessor[Client, Resource, Apply],
	apply Apply,
) *podder[Client, Resource, Apply] {
	m, o := acc.applyMeta(&apply)
	if internal.ValueOrZero(m.Kind) == "" {
		panic(fmt.Errorf("require TypeMeta.Name for %T", apply))
	}
	if internal.ValueOrZero(o.Name) == "" {
		panic(fmt.Errorf("require ObjectMeta.Name for %s", *m.Kind))
	}
	// TODO: add standard annotations and labels
	return &podder[Client, Resource, Apply]{acc, apply}
}

// ID implements resource.Resource.
func (p *podder[Client, Resource, Apply]) ID() string {
	m, o := p.acc.applyMeta(&p.apply)
	return "k8s/" + *m.Kind + "/" + *o.Name
}

// Start implements resource.Resource.
func (p *podder[Client, Resource, Apply]) Start(ctx *resource.Context) error {
	sc := p.client(ctx)
	// TODO: preserve scale settings if the resource already exists
	if _, err := sc.Apply(ctx, &p.apply, applyOpts(ctx)); err != nil {
		m, o := p.acc.applyMeta(&p.apply)
		return fmt.Errorf("failed to apply %s %s: %w", *m.Kind, *o.Name, err)
	}
	return nil
}

// Stop implements resource.Resource.
func (p *podder[Client, Resource, Apply]) Stop(ctx *resource.Context) error {
	sc := p.client(ctx)
	m, o := p.acc.applyMeta(&p.apply)
	if err := sc.Delete(ctx, *o.Name, deleteOpts(ctx)); err != nil {
		return fmt.Errorf("failed to delete %s %s: %w", *m.Kind, *o.Name, err)
	}
	return nil
}

// Ready implements resource.Resource.
func (p *podder[Client, Resource, Apply]) Ready(ctx *resource.Context) (bool, error) {
	panic("unimplemented")
}

// ContainerImages implements resource.ContainerResource.
func (p *podder[Client, Resource, Apply]) ContainerImages(ctx *resource.Context) ([]string, error) {
	pt := p.acc.podTemplate(&p.apply)
	// TODO: de-dupe
	ret := make([]string, 0, len(pt.InitContainers)+len(pt.Containers))
	for _, ic := range pt.InitContainers {
		if ic.Image != nil {
			ret = append(ret, *ic.Image)
		}
	}
	for _, c := range pt.Containers {
		if c.Image != nil {
			ret = append(ret, *c.Image)
		}
	}
	return ret, nil
}

func (p *podder[Client, Resource, Apply]) client(ctx *resource.Context) Client {
	return p.acc.getClient(
		resource.ContextValue[kubernetes.Interface](ctx),
		resource.ContextValue[Namespace](ctx),
	)
}

func StatefulSet(apply applyAppsV1.StatefulSetApplyConfiguration) resource.ContainerResource {
	addon.CheckInitialized()
	return newPodder(accStatefulSet, apply)
}

func Deployment(apply applyAppsV1.DeploymentApplyConfiguration) resource.ContainerResource {
	addon.CheckInitialized()
	return newPodder(accDeployment, apply)
}
