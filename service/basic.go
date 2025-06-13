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

func New(
	name string,
	opts ...basicOpt,
) Service {
	if strings.ContainsFunc(name, unicode.IsSpace) {
		panic(fmt.Errorf("service name %q must not contain whitespace", name))
	}
	bs := &basicService{}
	bs.name = name
	svc := Service(bs)
	for _, o := range opts {
		svc = o(svc, bs)
	}
	// validate
	if len(bs.resources) == 0 {
		panic(fmt.Errorf("service %s needs some resources", name))
	}
	return svc
}

type basicOpt func(Service, *basicService) Service

func WithResources(resources ...resource.Resource) basicOpt {
	return func(svc Service, bs *basicService) Service {
		bs.resources = append(bs.resources, func(context.Context) []resource.Resource {
			return resources
		})
		return svc
	}
}

func WithResourceFuncs(funcs ...func(context.Context) []resource.Resource) basicOpt {
	return func(svc Service, bs *basicService) Service {
		bs.resources = append(bs.resources, funcs...)
		return svc
	}
}
