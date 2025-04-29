package k8s

import (
	apiCoreV1 "k8s.io/api/core/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	clientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"fastcat.org/go/gdev/resource"
)

type pvc struct {
	appliable[
		clientCoreV1.PersistentVolumeClaimInterface,
		apiCoreV1.PersistentVolumeClaim,
		*applyCoreV1.PersistentVolumeClaimApplyConfiguration,
	]
}

func (r *pvc) Stop(ctx *resource.Context) error {
	// deleting PVCs will generally result in deleting storage, which we do not
	// want (e.g. deleting DB data). This is therefore a no-op.
	return nil
}

func PersistentVolumeClaim(apply *applyCoreV1.PersistentVolumeClaimApplyConfiguration) Resource {
	return &pvc{newApply(accPVC, apply)}
}
