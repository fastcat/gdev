package service

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"fastcat.org/go/gdev/resource"
)

type basicService struct {
	name      string
	resources []func(context.Context) []resource.Resource
}

var _ Service = (*basicService)(nil)

// Name implements Service.
func (s *basicService) Name() string {
	return s.name
}

// Resources implements Service.
func (s *basicService) Resources(ctx context.Context) []resource.Resource {
	ret := make([]resource.Resource, 0, len(s.resources))
	for _, r := range s.resources {
		ret = append(ret, r(ctx)...)
	}
	return ret
}

func NewService(
	name string,
	opts ...basicOpt,
) *basicService {
	if strings.ContainsFunc(name, unicode.IsSpace) {
		panic(fmt.Errorf("service name %q must not contain whitespace", name))
	}
	svc := &basicService{name: name}
	for _, o := range opts {
		o(svc)
	}
	// validate
	if len(svc.resources) == 0 {
		panic(fmt.Errorf("service %s needs some resources", name))
	}
	return svc
}

type basicOpt func(*basicService)

func WithResources(resources ...resource.Resource) basicOpt {
	return func(bs *basicService) {
		bs.resources = append(bs.resources, func(context.Context) []resource.Resource {
			return resources
		})
	}
}

func WithResourceFuncs(funcs ...func(context.Context) []resource.Resource) basicOpt {
	return func(bs *basicService) {
		bs.resources = append(bs.resources, funcs...)
	}
}
