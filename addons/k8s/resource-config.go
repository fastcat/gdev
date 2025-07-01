package k8s

import (
	apiCoreV1 "k8s.io/api/core/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	clientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"fastcat.org/go/gdev/addons/containers"
)

type configMap struct {
	appliable[
		clientCoreV1.ConfigMapInterface,
		apiCoreV1.ConfigMap,
		*applyCoreV1.ConfigMapApplyConfiguration,
	]
}

func ConfigMap(apply *applyCoreV1.ConfigMapApplyConfiguration) Resource {
	l := containers.DefaultLabels()
	apply.
		WithLabels(l).
		WithAnnotations(l)
	return &configMap{newApply(accConfigMap, apply)}
}
