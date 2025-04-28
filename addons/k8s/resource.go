package k8s

import (
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func applyOpts(*resource.Context) apiMetaV1.ApplyOptions {
	return apiMetaV1.ApplyOptions{
		Force:        true,
		FieldManager: instance.AppName(),
		// TODO: dry run
	}
}

func deleteOpts(*resource.Context) apiMetaV1.DeleteOptions {
	return apiMetaV1.DeleteOptions{
		PropagationPolicy: internal.Ptr(apiMetaV1.DeletePropagationBackground),
	}
}
