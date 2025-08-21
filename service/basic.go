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
	hasModal  map[Mode]bool
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

func (s *basicService) HasModal(mode Mode) bool {
	return mode != ModeDisabled && s.hasModal[mode]
}

func New(
	name string,
	opts ...BasicOpt,
) Service {
	if strings.ContainsFunc(name, unicode.IsSpace) {
		panic(fmt.Errorf("service name %q must not contain whitespace", name))
	}
	bs := &basicService{
		name:     name,
		hasModal: make(map[Mode]bool),
	}
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

type BasicOpt func(Service, *basicService) Service

func WithResources(resources ...resource.Resource) BasicOpt {
	return func(svc Service, bs *basicService) Service {
		bs.hasModal[ModeDefault] = true
		bs.resources = append(bs.resources, func(context.Context) []resource.Resource {
			return resources
		})
		return svc
	}
}

func WithResourceFuncs(funcs ...func(context.Context) []resource.Resource) BasicOpt {
	return func(svc Service, bs *basicService) Service {
		bs.hasModal[ModeDefault] = true
		bs.resources = append(bs.resources, funcs...)
		return svc
	}
}

// WithModalResources adds resources that are only used in a specific mode.
//
// If the services is started in any other mode, these will be converted to Anti
// resources and stopped during stack start.
func WithModalResources(
	mode Mode,
	resources ...resource.Resource,
) BasicOpt {
	return WithModalResourceFuncs(mode, func(ctx context.Context) []resource.Resource { return resources })
}

func WithModalResourceFuncs(
	mode Mode,
	funcs ...func(context.Context) []resource.Resource,
) BasicOpt {
	if !mode.Valid() || mode == ModeDisabled {
		panic(fmt.Errorf("invalid mode %s for modal resources", mode))
	}
	return func(svc Service, bs *basicService) Service {
		bs.hasModal[mode] = true
		bs.resources = append(bs.resources, func(ctx context.Context) []resource.Resource {
			m, _ := ServiceMode(ctx, svc.Name())
			ret := make([]resource.Resource, 0, len(funcs))
			for _, f := range funcs {
				for _, r := range f(ctx) {
					// convert to anti resources if the mode doesn't match
					if m != mode && !resource.IsAnti(r) {
						r = resource.Anti(r)
					}
					ret = append(ret, r)
				}
			}
			return ret
		})
		return svc
	}
}
