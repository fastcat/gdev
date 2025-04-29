package k8s

import (
	apiCoreV1 "k8s.io/api/core/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	clientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"fastcat.org/go/gdev/resource"
)

type service struct {
	appliable[
		clientCoreV1.ServiceInterface,
		apiCoreV1.Service,
		*applyCoreV1.ServiceApplyConfiguration,
	]
}

// Ready implements resource.Resource.
func (s *service) Ready(ctx *resource.Context) (bool, error) {
	// services have no ready gates
	return true, nil
}

func Service(apply *applyCoreV1.ServiceApplyConfiguration) Resource {
	addon.CheckInitialized()
	return &service{newApply(accService, apply)}
}
