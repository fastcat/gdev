package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"fastcat.org/go/gdev/resource"
)

type basicService struct {
	name      string
	resources []func(context.Context) []resource.Resource
}

type basicServiceWithSource struct {
	basicService
	localSource  func(context.Context) (root, subDir string, err error)
	remoteSource func(context.Context) (vcs, repo string, err error)
}

var (
	_ Service           = (*basicService)(nil)
	_ ServiceWithSource = (*basicServiceWithSource)(nil)
)

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
) Service {
	if strings.ContainsFunc(name, unicode.IsSpace) {
		panic(fmt.Errorf("service name %q must not contain whitespace", name))
	}
	svc := &basicServiceWithSource{}
	svc.name = name
	for _, o := range opts {
		o(svc)
	}
	// validate
	if len(svc.resources) == 0 {
		panic(fmt.Errorf("service %s needs some resources", name))
	}
	if svc.localSource == nil && svc.remoteSource == nil {
		// TODO: this is stupid
		return &svc.basicService
	}
	return svc
}

type basicOpt func(*basicServiceWithSource)

func WithResources(resources ...resource.Resource) basicOpt {
	return func(bs *basicServiceWithSource) {
		bs.resources = append(bs.resources, func(context.Context) []resource.Resource {
			return resources
		})
	}
}

func WithResourceFuncs(funcs ...func(context.Context) []resource.Resource) basicOpt {
	return func(bs *basicServiceWithSource) {
		bs.resources = append(bs.resources, funcs...)
	}
}

func WithLocalSource(
	root, subDir string,
) basicOpt {
	return func(bs *basicServiceWithSource) {
		if root == "" {
			panic(fmt.Errorf("local source root must not be empty"))
		}
		if subDir == "" {
			subDir = "."
		}
		bs.localSource = func(context.Context) (string, string, error) {
			return root, subDir, nil
		}
	}
}

func WithLocalSourceFunc(
	fn func(context.Context) (root, subDir string, err error),
) basicOpt {
	return func(bs *basicServiceWithSource) {
		if fn == nil {
			panic(fmt.Errorf("local source function must not be nil"))
		}
		bs.localSource = fn
	}
}

var ErrNoLocalSource = errors.New("does not have a local source")

func (s *basicServiceWithSource) LocalSource(ctx context.Context) (root, subDir string, err error) {
	if s.localSource == nil {
		return "", "", fmt.Errorf("service %s %w", s.name, ErrNoLocalSource)
	}
	return s.localSource(ctx)
}

func WithRemoteSource(
	vcs, repo string,
) basicOpt {
	return func(bs *basicServiceWithSource) {
		if vcs == "" || repo == "" {
			panic(fmt.Errorf("remote source vcs and repo must not be empty"))
		}
		bs.remoteSource = func(context.Context) (string, string, error) {
			return vcs, repo, nil
		}
	}
}

func WithRemoteSourceFunc(
	fn func(context.Context) (vcs, repo string, err error),
) basicOpt {
	return func(bs *basicServiceWithSource) {
		if fn == nil {
			panic(fmt.Errorf("remote source function must not be nil"))
		}
		bs.remoteSource = fn
	}
}

var ErrNoRemoteSource = errors.New("does not have a remote source")

func (s *basicServiceWithSource) RemoteSource(ctx context.Context) (vcs, repo string, err error) {
	if s.remoteSource == nil {
		return "", "", fmt.Errorf("service %s %w", s.name, ErrNoRemoteSource)
	}
	return s.remoteSource(ctx)
}
