package service

import "context"

type serviceWithSource struct {
	Service
	localSource  func(context.Context) (root, subDir string, err error)
	remoteSource func(context.Context) (vcs, repo string, err error)
}

var _ ServiceWithSource = (*serviceWithSource)(nil)

func WithSource(
	localRoot, localSubDir string,
	remoteVCS, remoteRepo string,
) basicOpt {
	return WithSourceFuncs(
		func(context.Context) (string, string, error) {
			return localRoot, localSubDir, nil
		},
		func(context.Context) (string, string, error) {
			return remoteVCS, remoteRepo, nil
		},
	)
}

func WithSourceFuncs(
	localSourceFunc func(context.Context) (root, subDir string, err error),
	remoteSourceFunc func(context.Context) (vcs, repo string, err error),
) basicOpt {
	return func(svc Service, bs *basicService) Service {
		return &serviceWithSource{
			Service:      svc,
			localSource:  localSourceFunc,
			remoteSource: remoteSourceFunc,
		}
	}
}

func (s *serviceWithSource) LocalSource(ctx context.Context) (root, subDir string, err error) {
	return s.localSource(ctx)
}

func (s *serviceWithSource) RemoteSource(ctx context.Context) (vcs, repo string, err error) {
	return s.remoteSource(ctx)
}

func (s *serviceWithSource) UsesSourceInMode(mode Mode) bool {
	// by default assume debug mode will be built by the debug launch, and default
	// mode uses docker/etc artifacts instead of any source code.
	return mode == ModeLocal
}
