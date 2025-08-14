package k3s

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"slices"
	"strings"

	"fastcat.org/go/gdev/addons/github"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/shx"
	"fastcat.org/go/gdev/sys"
)

const DefaultInstallPath = "/usr/local/bin/k3s"

func InstallStable(ctx context.Context, path string) error {
	if path == "" {
		path = DefaultInstallPath
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

	if installed, _ := InstalledVersion(ctx, path); ver == installed {
		// TODO: verify checksum
		return nil
	}

	ghc := github.NewClient()
	rel, err := ghc.GetRelease(ctx, "k3s-io", "k3s", ver)
	if err != nil {
		return err
	}
	assetName := "k3s"
	if runtime.GOARCH != "amd64" {
		assetName += "-" + runtime.GOARCH
	}
	i := slices.IndexFunc(rel.Assets, func(a github.ReleaseAsset) bool { return a.Name == assetName })
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
	if _, err := shx.Run(
		ctx,
		[]string{"install", tfn, path},
		shx.WithSudo("install k3s"),
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
	if err != nil {
		return "", err
	}
	// expected output format:
	//     k3s version v1.31.5+k3s1 (56ec5dd4)
	//     go version go1.22.10
	for l := range strings.Lines(string(data)) {
		if !strings.HasPrefix(l, "k3s version ") {
			continue
		} else if f := strings.Fields(l); len(f) >= 3 {
			return f[2], nil
		}
	}
	return "", nil
}

func InstallSudoers(ctx context.Context, path string) error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	content := fmt.Sprintf(
		"# THIS FILE IS GENERATED, DO NOT EDIT\n"+
			"%[2]s ALL=(ALL:ALL) NOPASSWD: %[1]s\n"+
			"%[2]s ALL=(ALL:ALL) NOPASSWD: /usr/bin/pkill -TERM k3s, /bin/pkill -TERM k3s\n"+
			"%[2]s ALL=(ALL:ALL) NOPASSWD: /usr/bin/cat /etc/rancher/k3s/k3s.yaml, "+
			"/bin/cat /etc/rancher/k3s/k3s.yaml\n",
		path,
		u.Username,
	)
	// don't overwrite unnecessarily
	existing, err := sys.ReadFileAsRoot(ctx, path, true)
	if err == nil && content == string(existing) {
		// nothing to do
		return nil
	}
	if err := sys.WriteFileAsRoot(
		ctx,
		fmt.Sprintf("/etc/sudoers.d/%s-k3s", instance.AppName()),
		strings.NewReader(content),
		0o444,
	); err != nil {
		return err
	}

	return nil
}
