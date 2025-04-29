package k8s

import (
	"fastcat.org/go/gdev/resource"
)

type k8sResource interface {
	K8SKind() string
	K8SNamespace() string
	K8SName() string
}

type Resource interface {
	resource.Resource
	k8sResource
}

type ContainerResource interface {
	resource.ContainerResource
	k8sResource
}
