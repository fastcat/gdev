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
	name         string
	resources    []func(context.Context) []resource.Resource
	localSource  func(context.Context) (root, subDir string, err error)
	remoteSource func(context.Context) (vcs, repo string, err error)
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

func WithLocalSource(
	root, subDir string,
) basicOpt {
	return func(bs *basicService) {
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
	return func(bs *basicService) {
		if fn == nil {
			panic(fmt.Errorf("local source function must not be nil"))
		}
		bs.localSource = fn
	}
}

var ErrNoLocalSource = errors.New("does not have a local source")

func (s *basicService) LocalSource(ctx context.Context) (root, subDir string, err error) {
	if s.localSource == nil {
		return "", "", fmt.Errorf("service %s %w", s.name, ErrNoLocalSource)
	}
	return s.localSource(ctx)
}

func WithRemoteSource(
	vcs, repo string,
) basicOpt {
	return func(bs *basicService) {
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
	return func(bs *basicService) {
		if fn == nil {
			panic(fmt.Errorf("remote source function must not be nil"))
		}
		bs.remoteSource = fn
	}
}

var ErrNoRemoteSource = errors.New("does not have a remote source")

func (s *basicService) RemoteSource(ctx context.Context) (vcs, repo string, err error) {
	if s.remoteSource == nil {
		return "", "", fmt.Errorf("service %s %w", s.name, ErrNoRemoteSource)
	}
	return s.remoteSource(ctx)
}
