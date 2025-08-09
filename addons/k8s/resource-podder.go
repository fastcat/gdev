package k8s

import (
	"context"

	applyAppsV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applyBatchV1 "k8s.io/client-go/applyconfigurations/batch/v1"

	"fastcat.org/go/gdev/addons/containers"
)

// podder generalizes the pattern of a k8s resource that schedules pods
type podder[
	Client client[Resource, Apply],
	Resource any,
	Apply apply[Apply],
] struct {
	appliable[Client, Resource, Apply]
}

func newPodder[
	Client client[Resource, Apply],
	Resource any,
	Apply apply[Apply],
](
	acc accessor[Client, Resource, Apply],
	apply Apply,
) *podder[Client, Resource, Apply] {
	// TODO: add standard annotations and labels
	return &podder[Client, Resource, Apply]{newAppliable(acc, apply)}
}

// ContainerImages implements resource.ContainerResource.
func (p *podder[Client, Resource, Apply]) ContainerImages(ctx context.Context) ([]string, error) {
	pt := p.acc.podTemplate(p.apply)
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

func StatefulSet(apply *applyAppsV1.StatefulSetApplyConfiguration) ContainerResource {
	l := containers.DefaultLabels()
	apply.
		WithLabels(l).
		WithAnnotations(l)
	apply.Spec.Template.
		WithLabels(l).
		WithAnnotations(l)
	return newPodder(accStatefulSet, apply)
}

func Deployment(apply *applyAppsV1.DeploymentApplyConfiguration) ContainerResource {
	l := containers.DefaultLabels()
	apply.
		WithLabels(l).
		WithAnnotations(l)
	apply.Spec.Template.
		WithLabels(l).
		WithAnnotations(l)
	return newPodder(accDeployment, apply)
}

func CronJob(apply *applyBatchV1.CronJobApplyConfiguration) ContainerResource {
	l := containers.DefaultLabels()
	apply.
		WithLabels(l).
		WithAnnotations(l)
	apply.Spec.JobTemplate.Spec.Template.
		WithLabels(l).
		WithAnnotations(l)
	return newPodder(accCronJob, apply)
}

func BatchJob(apply *applyBatchV1.JobApplyConfiguration) ContainerResource {
	l := containers.DefaultLabels()
	apply.
		WithLabels(l).
		WithAnnotations(l)
	apply.Spec.Template.
		WithLabels(l).
		WithAnnotations(l)
	return newPodder(accBatchJob, apply)
}

// NOTE: Direct Pod manipulation is intentionally left out because it's a Bad Idea
