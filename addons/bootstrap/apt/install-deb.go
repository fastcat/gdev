package apt

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/jedib0t/go-pretty/v6/progress"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/lib/httpx"
)

func InstallDownloadedPackage(
	ctx *bootstrap.Context,
	name string,
	src PackageSource,
	opts *InstallPackageOptions,
) error {
	if opts == nil {
		opts = &InstallPackageOptions{}
	}

	skip := false
	if installed, err := DpkgInstalled(ctx); err != nil {
		return err
	} else if _, ok := installed[name]; ok {
		skip = true
		if opts.AlwaysInstall {
			skip = false
		} else if opts.ForceInstall != nil {
			if force, err := opts.ForceInstall(ctx); err != nil {
				return err
			} else if force {
				skip = true
			}
		}
	}
	if skip {
		fmt.Printf("Skip: package %q already installed\n", name)
		return nil
	}

	stream, size, err := src.Acquire(ctx)
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

type PackageSource interface {
	Acquire(*bootstrap.Context) (io.ReadCloser, int64, error)
}

type HTTPPackageSource func(*bootstrap.Context) (*http.Request, error)

// Acquire implements PackageSource.
func (h HTTPPackageSource) Acquire(ctx *bootstrap.Context) (io.ReadCloser, int64, error) {
	req, err := h(ctx)
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

// URLPackageSource is a general purpose package source for doing an
// unauthenticated GET from the given URL.
//
// NOTE: this does not work for GitHub asset downloads. Use
// [github.PackageSource] for that.
func URLPackageSource(uri string) PackageSource {
	return HTTPPackageSource(func(ctx *bootstrap.Context) (*http.Request, error) {
		return http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			uri,
			nil,
		)
	})
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
	wg.Go(pw.Render)
	defer wg.Wait()
	defer pw.Stop()

	_, err := io.Copy(&progressWriter{pt, dest}, src)
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
