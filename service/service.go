package service

import (
	"context"

	"fastcat.org/go/gdev/resource"
)

type Service interface {
	Name() string
	Resources(context.Context) []resource.Resource
	LocalSource(context.Context) (root, subDir string, err error)
	RemoteSource(context.Context) (vcs, repo string, err error)
}
