package k8s

import (
	"sync"

	metaApplyV1 "k8s.io/client-go/applyconfigurations/meta/v1"

	"fastcat.org/go/gdev/addons/containers"
	"fastcat.org/go/gdev/instance"
)

var AppLabel = sync.OnceValue(func() string {
	return "fastcat.org/go/gdev/" + instance.AppName() + "/app"
})

func AppSelector(name string) *metaApplyV1.LabelSelectorApplyConfiguration {
	return metaApplyV1.LabelSelector().
		WithMatchLabels(map[string]string{
			AppLabel(): name,
		})
}

func AppLabels(name string) map[string]string {
	m := containers.DefaultLabels()
	m[AppLabel()] = name
	return m
}
