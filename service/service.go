package service

import (
	"context"

	"fastcat.org/go/gdev/resource"
)

type Service interface {
	Name() string
	Resources(context.Context) []resource.Resource
	// LocalSource returns where the source code for this service is located on
	// the local system, or if not present where it _should_ be located (so
	// RemoteSource can be used to clone it).
	LocalSource(context.Context) (root, subDir string, err error)
	// RemoteSource returns the VCS and repository URL for this service's source
	// code.
	RemoteSource(context.Context) (vcs, repo string, err error)
}
