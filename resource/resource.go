package resource

import "context"

type Resource interface {
	ID() string
	Start(context.Context) error
	Stop(context.Context) error
	Ready(context.Context) (bool, error) // TODO: provide not-ready details
}

type ContainerResource interface {
	Resource
	ContainerImages(context.Context) ([]string, error)
}
