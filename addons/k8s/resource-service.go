package k8s

import (
	apiCoreV1 "k8s.io/api/core/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	clientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"fastcat.org/go/gdev/addons/containers"
)

type service struct {
	appliable[
		clientCoreV1.ServiceInterface,
		apiCoreV1.Service,
		*applyCoreV1.ServiceApplyConfiguration,
	]
}

func Service(apply *applyCoreV1.ServiceApplyConfiguration) Resource {
	l := containers.DefaultLabels()
	apply.
		WithLabels(l).
		WithAnnotations(l)
	return &service{newAppliable(accService, apply)}
}
