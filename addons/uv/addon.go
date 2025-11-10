package uv

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/github"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/lib/shx"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "uv",
		Description: func() string {
			return "uv python environment manager"
		},
		// Initialize: initialize, // initialized below to avoid circular dependency
	},
	Config: config{
		// Initialize your addon configuration here
	},
}

func init() {
	addon.Definition.Initialize = initialize
}

type config struct{}

type option func(*config)

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	configureBootstrap()

	addon.RegisterIfNeeded()
}

func initialize() error {
	return nil
}

var configureBootstrap = sync.OnceFunc(func() {
	bootstrap.Configure(bootstrap.WithSteps(
		bootstrap.NewStep(
			"Install uv",
			installUV,
			bootstrap.SimFunc(simUV),
		)),
	)
})

func installUV(ctx *bootstrap.Context) error {
	fmt.Println("Installing uv python environment manager...")

	ghc, rel, destDir, alreadyInstalled, err := prepUV(ctx)
	if err != nil {
		return err
	}

	// ~/.local/bin might not be in the PATH because most bashrc setups only add
	// it if it exists. Make sure it's there now so we can run the just-installed
	// copy.
	// TODO: put this in shx or something
	if slices.Index(filepath.SplitList(os.Getenv("PATH")), destDir) < 0 {
		if err := os.Setenv("PATH", destDir+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
			return fmt.Errorf("failed to add %s to PATH: %w", destDir, err)
		}
	}

	if alreadyInstalled {
		return nil
	}

	// e.g. uv-x86_64-unknown-linux-gnu.tar.gz
	// TODO: this is wrong for anything other than amd64 & arm64 linux
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	}
	baseName := "uv-" + arch + "-unknown-" + runtime.GOOS + "-gnu"
	assetName := baseName + ".tar.gz"
	i := slices.IndexFunc(rel.Assets, func(a github.ReleaseAsset) bool { return a.Name == assetName })
	if i < 0 {
		return fmt.Errorf("uv release %s has no asset %s", rel.TagName, assetName)
	}
	resp, err := ghc.Download(ctx, rel.Assets[i].URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	// /tmp is often a different filesystem from $HOME, preventing renames at the
	// end, so store this in the dest dir instead
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create asdf destination directory %s: %w", destDir, err)
	}
	tmpUV, err := os.CreateTemp(destDir, instance.AppName()+"-uv-*")
	if err != nil {
		return err
	}
	defer tmpUV.Close()           // nolint:errcheck
	defer os.Remove(tmpUV.Name()) // nolint:errcheck
	tmpUVX, err := os.CreateTemp(destDir, instance.AppName()+"-uvx-*")
	if err != nil {
		return err
	}
	defer tmpUVX.Close()           // nolint:errcheck
	defer os.Remove(tmpUVX.Name()) // nolint:errcheck

	// expect a tar.gz file with exactly two files in it named uv and uvx, in a
	// directory named for the tarball
	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("uv download %s corrupt: %w", rel.Assets[i].URL, err)
	}
	tr := tar.NewReader(zr)

	didUV, didUVX := false, false
	for {
		th, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("uv download %s corrupt: %w", rel.Assets[i].URL, err)
		}
		var f *os.File
		var did *bool
		discard := false
		switch th.Name {
		case baseName + "/":
			discard = true
		case path.Join(baseName, "uv"):
			f, did = tmpUV, &didUV
		case path.Join(baseName, "uvx"):
			f, did = tmpUVX, &didUVX
		}
		if discard {
			// directory entry to skip over
			if _, err := io.Copy(io.Discard, tr); err != nil {
				return fmt.Errorf("uv download %s corrupt: %w", rel.Assets[i].URL, err)
			}
			continue
		}
		if f == nil {
			return fmt.Errorf(
				"uv download %s corrupt: expected two files named 'uv' and 'uvx', got %s",
				rel.Assets[i].URL,
				th.Name,
			)
		}
		if *did {
			return fmt.Errorf(
				"uv download %s corrupt: duplicate file %s",
				rel.Assets[i].URL,
				th.Name,
			)
		}
		if _, err := io.Copy(f, tr); err != nil {
			return fmt.Errorf("failed to extract %s binary: %w", th.Name, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("failed to flush %s temp file %s: %w", th.Name, f.Name(), err)
		}
		if err := os.Chmod(f.Name(), 0o755); err != nil {
			return fmt.Errorf("failed to make %s executable: %w", th.Name, err)
		}
		*did = true
	}

	if !didUV || !didUVX {
		return fmt.Errorf(
			"uv download %s corrupt: expected two files named 'uv' and 'uvx', got uv=%t, uvx=%t",
			rel.Assets[i].URL,
			didUV,
			didUVX,
		)
	}

	// TODO: this assumes that ~/.local/bin/ is in the user's PATH, or at least
	// will be once they reboot if it wasn't added because it didn't exist yet.
	if err := os.Rename(tmpUV.Name(), filepath.Join(destDir, "uv")); err != nil {
		return fmt.Errorf("failed to install uv binary to %s: %w", destDir, err)
	}
	if err := os.Rename(tmpUVX.Name(), filepath.Join(destDir, "uvx")); err != nil {
		return fmt.Errorf("failed to install uvx binary to %s: %w", destDir, err)
	}

	return nil
}

func simUV(ctx *bootstrap.Context) error {
	fmt.Println("Simulating uv installation...")
	_, rel, _, alreadyInstalled, err := prepUV(ctx)
	if err != nil {
		return err
	} else if alreadyInstalled {
		return nil
	}
	fmt.Printf("Would install uv %s\n", rel.TagName)
	return nil
}

func prepUV(ctx *bootstrap.Context) (*github.Client, *github.Release, string, bool, error) {
	ghc := github.NewClient()
	rel, err := ghc.GetRelease(ctx, "astral-sh", "uv", "latest")
	if err != nil {
		return nil, nil, "", false, fmt.Errorf("failed to fetch uv release: %w", err)
	}

	destDir := filepath.Join(shx.HomeDir(), ".local", "bin")

	// check what version is installed, only update if it's different
	outdated := false
	for _, bin := range []string{"uv", "uvx"} {
		if res, err := shx.Run(ctx,
			[]string{filepath.Join(destDir, bin), "--version"},
			shx.CaptureOutput(),
		); err != nil {
			if !errors.Is(err, exec.ErrNotFound) {
				return nil, nil, "", false, err
			}
			outdated = true
			break
		} else if err := res.Err(); err != nil {
			// binary broken? force reinstallation
			outdated = true
			break
		} else if out, err := io.ReadAll(res.Stdout()); err != nil {
			// should not happen?
			return nil, nil, "", false, fmt.Errorf("failed to read `%s --version` output: %w", bin, err)
		} else if outStr := strings.TrimSpace(string(out)); outStr != bin+" "+rel.TagName {
			outdated = true
			break
		}
	}
	if !outdated {
		fmt.Printf("uv %s already installed, skipping\n", rel.TagName)
		return nil, nil, "", true, nil
	}
	return ghc, rel, destDir, false, nil
}
