package service

import (
	"context"

	"fastcat.org/go/gdev/resource"
)

type Service interface {
	Name() string
	Resources(context.Context) []resource.Resource
}
