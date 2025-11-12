package apt

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/jedib0t/go-pretty/v6/progress"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt/dpkg"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/lib/httpx"
)

func InstallDownloadedPackageStep[R PackageRelease](
	name string,
	src PackageSource[R],
	opts *InstallPackageOptions,
) *bootstrap.Step {
	return bootstrap.NewStep(
		"Install "+name,
		func(ctx *bootstrap.Context) error {
			return InstallDownloadedPackage(ctx, name, src, opts)
		},
		bootstrap.SimFunc(func(ctx *bootstrap.Context) error {
			return SimDownloadedPackage(ctx, name, src, opts)
		}),
	)
}

func InstallDownloadedPackage[R PackageRelease](
	ctx *bootstrap.Context,
	name string,
	src PackageSource[R],
	opts *InstallPackageOptions,
) error {
	opts, rel, skip, err := prepDownloadedPackage(ctx, name, src, opts)
	if err != nil {
		return err
	} else if skip {
		return nil
	}

	stream, size, err := src.Acquire(ctx, rel)
	if err != nil {
		return err
	}
	defer stream.Close() //nolint:errcheck

	tf, err := os.CreateTemp("", instance.AppName()+"-"+name+"-*.deb")
	if err != nil {
		return err
	}
	defer os.Remove(tf.Name()) // nolint:errcheck
	defer tf.Close()           // nolint:errcheck

	if err := streamWithProgress(stream, tf, "Downloading "+name+".deb", size); err != nil {
		return err
	} else if err := tf.Close(); err != nil {
		return err
	}

	sudoPrompt := opts.SudoPrompt
	if sudoPrompt == "" {
		sudoPrompt = "install " + name
	}
	return DoInstall(
		ctx,
		opts.ExtraAptOpts,
		[]string{tf.Name()},
		sudoPrompt,
	)
}

func SimDownloadedPackage[R PackageRelease](
	ctx *bootstrap.Context,
	name string,
	src PackageSource[R],
	opts *InstallPackageOptions,
) error {
	_, rel, skip, err := prepDownloadedPackage(ctx, name, src, opts)
	if err != nil {
		return err
	} else if skip {
		return nil
	}

	fmt.Printf("Would install package %q version %s\n", name, rel.PackageVersion())
	return nil
}

func prepDownloadedPackage[R PackageRelease](
	ctx *bootstrap.Context,
	name string,
	src PackageSource[R],
	opts *InstallPackageOptions,
) (*InstallPackageOptions, R, bool, error) {
	if opts == nil {
		opts = &InstallPackageOptions{}
	}

	rel, err := src.Prepare(ctx)
	if err != nil {
		return opts, rel, false, err
	}

	skip := false
	if installed, err := DpkgInstalled(ctx); err != nil {
		return opts, rel, false, err
	} else if installedVer, ok := installed[name]; ok {
		skip = true
		if opts.AlwaysInstall {
			skip = false
		} else if opts.ForceInstall != nil {
			if force, err := opts.ForceInstall(ctx); err != nil {
				return opts, rel, true, err
			} else if force {
				skip = false
			}
		}
		// don't skip if the available version is newer than the installed version
		if skip {
			if relVer := rel.PackageVersion(); relVer != "" && relVer != installedVer {
				// also treat parse errors on the versions as a reason to not skip
				if cmp, err := dpkg.Compare(relVer, installedVer); err == nil && cmp > 0 {
					skip = false
				}
			}
		}
	}
	if skip {
		fmt.Printf("Skip: package %q version %s already installed\n", name, rel.PackageVersion())
		return opts, rel, true, nil
	}
	return opts, rel, false, nil
}

type InstallPackageOptions struct {
	// If AlwaysInstall is true, the package will always be downloaded and
	// installed. If false (default), it will only be installed if a package of
	// the same is not already installed (without any version checking).
	AlwaysInstall bool
	// If ForceInstall is set, it will be called to determine whether to install
	// the package if it is already installed. If AlwaysInstall is set true, this
	// field is ignored.
	ForceInstall func(*bootstrap.Context) (bool, error)
	// Prompt for invoking sudo to install the package. If empty (default), a
	// prompt using the package name will be generated.
	SudoPrompt   string
	ExtraAptOpts []string
}

type PackageRelease interface {
	PackageVersion() string
}

type StaticPackageRelease string

func (s StaticPackageRelease) PackageVersion() string { return string(s) }

type PackageSource[R PackageRelease] interface {
	Prepare(*bootstrap.Context) (R, error)
	Acquire(*bootstrap.Context, R) (io.ReadCloser, int64, error)
}

type HTTPPackageSource[R PackageRelease] struct {
	Rel func(*bootstrap.Context) (R, error)
	Req func(*bootstrap.Context, R) (*http.Request, error)
}

func (h *HTTPPackageSource[R]) Prepare(ctx *bootstrap.Context) (R, error) {
	return h.Rel(ctx)
}

// Acquire implements PackageSource.
func (h *HTTPPackageSource[R]) Acquire(ctx *bootstrap.Context, rel R) (io.ReadCloser, int64, error) {
	req, err := h.Req(ctx, rel)
	if err != nil {
		return nil, 0, err
	}
	if r, err := http.DefaultClient.Do(req); err != nil {
		return nil, 0, err
	} else if !httpx.IsHTTPOk(r) {
		// this will drain and close r.Body
		return nil, 0, httpx.HTTPResponseErr(r, fmt.Sprintf("failed to download %s", req.URL.String()))
	} else {
		return r.Body, r.ContentLength, nil
	}
}

var _ PackageSource[StaticPackageRelease] = (*HTTPPackageSource[StaticPackageRelease])(nil)

// URLPackageSource is a general purpose package source for doing an
// unauthenticated GET from the given URL.
//
// NOTE: this does not work for GitHub asset downloads. Use
// [github.PackageSource] for that.
func URLPackageSource(uri, ver string) *HTTPPackageSource[StaticPackageRelease] {
	return &HTTPPackageSource[StaticPackageRelease]{
		Rel: func(ctx *bootstrap.Context) (StaticPackageRelease, error) {
			return StaticPackageRelease(ver), nil
		},
		Req: func(ctx *bootstrap.Context, _ StaticPackageRelease) (*http.Request, error) {
			return http.NewRequestWithContext(
				ctx,
				http.MethodGet,
				uri,
				nil,
			)
		},
	}
}

func streamWithProgress(
	src io.Reader,
	dest io.Writer,
	msg string,
	size int64,
) error {
	// progress bar for download since it takes a moment
	pw := progress.NewWriter()
	// progress has an internal context, but doesn't support setting it to base
	// off something other than context.Background()
	pt := &progress.Tracker{
		Message: msg,
		Total:   size,
		Units:   progress.UnitsBytes,
	}
	pw.SetTrackerPosition(progress.PositionRight)
	pt.Start()
	pw.AppendTracker(pt)
	var wg sync.WaitGroup
	// if we run Render in the "background" there is a race where we might finish
	// our work before it initializes, and then Stop() won't actually stop it, so
	// we have to run Render in the "foreground" and the streaming in the
	// background.
	var err error
	wg.Go(func() {
		defer pw.Stop()
		_, err = io.Copy(&progressWriter{pt, dest}, src)
	})
	pw.Render()
	wg.Wait()
	return err
}

type progressWriter struct {
	t *progress.Tracker
	w io.Writer
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.w.Write(p)
	pw.t.Increment(int64(n))
	return n, err
}
