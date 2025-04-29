package k3s

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
)

const DefaultInstallPath = "/usr/local/bin/k3s"

func InstallStable(ctx context.Context, dest string) error {
	if dest == "" {
		dest = DefaultInstallPath
	}
	relData, err := getK3SChannels(ctx, nil)
	if err != nil {
		return err
	}
	stableData := relData.channel("stable")
	if stableData == nil {
		return fmt.Errorf("cannot find release information for k3s stable channel")
	}
	ver := stableData.Latest
	ghc := internal.NewGitHubClient()
	rel, err := ghc.Release(ctx, "k3s-io", "k3s", ver)
	if err != nil {
		return err
	}
	assetName := "k3s"
	if runtime.GOARCH != "amd64" {
		assetName += "-" + runtime.GOARCH
	}
	i := slices.IndexFunc(rel.Assets, func(a internal.GitHubReleaseAsset) bool { return a.Name == assetName })
	if i < 0 {
		return fmt.Errorf("k3s release %s has no asset %s", ver, assetName)
	}
	resp, err := ghc.Download(ctx, rel.Assets[i].URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	// download to a temp file
	tf, err := os.CreateTemp("", instance.AppName()+"-k3s-*")
	if err != nil {
		return err
	}
	tfn := tf.Name()
	defer os.Remove(tfn) // nolint:errcheck
	if _, err := io.Copy(tf, resp.Body); err != nil {
		return err
	} else if err := tf.Close(); err != nil {
		return err
	}

	// TODO: fetch the hash from github too and check it, reinstall or error if
	// checksum doesn't match?

	// install the downloaded file to the destination location
	if err := internal.Shell(
		ctx,
		[]string{"install", tfn, dest},
		internal.WithSudo("install k3s"),
	); err != nil {
		return err
	}

	return nil
}

func InstalledVersion(ctx context.Context, path string) (string, error) {
	if path == "" {
		path = DefaultInstallPath
	}
	cmd := exec.CommandContext(ctx, path, "--version")
	cmd.Dir = "/"
	data, err := cmd.Output()
	if err == nil {
		return "", err
	}
	// expected output format:
	//     k3s version v1.31.5+k3s1 (56ec5dd4)
	//     go version go1.22.10
	for l := range strings.Lines(string(data)) {
		if !strings.HasPrefix(l, "k3s version ") {
			continue
		} else if f := strings.Fields(l); len(f) >= 3 {
			return f[3], nil
		}
	}
	return "", nil
}
