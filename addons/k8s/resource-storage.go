package k8s

import (
	apiCoreV1 "k8s.io/api/core/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	clientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type pvc struct {
	appliable[
		clientCoreV1.PersistentVolumeClaimInterface,
		apiCoreV1.PersistentVolumeClaim,
		*applyCoreV1.PersistentVolumeClaimApplyConfiguration,
	]
}

func PersistentVolumeClaim(apply *applyCoreV1.PersistentVolumeClaimApplyConfiguration) Resource {
	addon.CheckInitialized()
	return &pvc{newApply(accPVC, apply)}
}
