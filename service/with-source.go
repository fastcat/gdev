package service

import "context"

type serviceWithSource struct {
	Service
	localSource  func(context.Context) (root, subDir string, err error)
	remoteSource func(context.Context) (vcs, repo string, err error)
}

var _ ServiceWithSource = (*serviceWithSource)(nil)

func WithSource(
	svc Service,
	localRoot, localSubDir string,
	remoteVCS, remoteRepo string,
) ServiceWithSource {
	return &serviceWithSource{
		Service: svc,
		localSource: func(context.Context) (string, string, error) {
			return localRoot, localSubDir, nil
		},
		remoteSource: func(context.Context) (string, string, error) {
			return remoteVCS, remoteRepo, nil
		},
	}
}

func WithSourceFuncs(
	svc Service,
	localSourceFunc func(context.Context) (root, subDir string, err error),
	remoteSourceFunc func(context.Context) (vcs, repo string, err error),
) ServiceWithSource {
	return &serviceWithSource{
		Service:      svc,
		localSource:  localSourceFunc,
		remoteSource: remoteSourceFunc,
	}
}

func (s *serviceWithSource) LocalSource(ctx context.Context) (root, subDir string, err error) {
	return s.localSource(ctx)
}

func (s *serviceWithSource) RemoteSource(ctx context.Context) (vcs, repo string, err error) {
	return s.remoteSource(ctx)
}
