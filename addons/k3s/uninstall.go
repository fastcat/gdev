package k3s

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"fastcat.org/go/gdev/lib/shx"
)

func Uninstall(ctx context.Context, dest string) error {
	// compare create_uninstall and create_killall in
	// https://github.com/k3s-io/k3s/blob/v1.33.2%2Bk3s1/install.sh#L874

	multiSudo := func(steps ...[]string) error {
		for _, step := range steps {
			fmt.Println("Running:", strings.Join(step[1:], " "))
			if res, err := shx.Run(
				ctx,
				step[1:],
				shx.PassOutput(),
				shx.WithSudo(step[0]),
			); err != nil {
				return fmt.Errorf("failed to %s: %w", step[0], err)
			} else if err := res.Err(); err != nil {
				// ignore exit errors
				fmt.Fprintf(os.Stderr, "WARNING: %s failed with error: %v\n", step[0], err)
			}
		}
		return nil
	}

	if err := multiSudo(
		[]string{"kill k3s", "pkill", "-TERM", "k3s"},
		// this is a lot easier than the ps tree solution the k3s script uses
		[]string{"stop pods", "systemctl", "stop", "kubepods.slice"},
	); err != nil {
		return err
	}

	mountsData, err := os.ReadFile("/proc/self/mounts")
	if err != nil {
		return err
	}
	mountLines := slices.Collect(strings.Lines(string(mountsData)))
	mountDirs := make([]string, 0, len(mountLines))
	for _, line := range mountLines {
		fields := strings.SplitN(line, " ", 3)
		if len(fields) < 3 {
			return fmt.Errorf("unexpected mount line: %q", strings.TrimSpace(line))
		}
		mountDirs = append(mountDirs, fields[1])
	}
	slices.Sort(mountDirs)
	slices.Reverse(mountDirs)
	unmountAndRemove := func(dir string) error {
		var dirPfx string
		if strings.HasSuffix(dir, "*") {
			dir = strings.TrimSuffix(dir, "*")
			dirPfx = dir
		} else {
			dirPfx = dir + string(filepath.Separator)
		}
		for _, mp := range mountDirs {
			if mp != dir && !strings.HasPrefix(mp, dirPfx) {
				continue
			}
			if err := multiSudo(
				[]string{"unmount " + mp, "umount", "-f", mp},
				[]string{"remove " + mp, "rm", "-rf", mp},
			); err != nil {
				return err
			}
		}
		return nil
	}
	for _, dir := range []string{
		"/run/k3s",
		// upstream only removes pods & plugins dirs, we want to wipe everything tho
		"/var/lib/kubelet",
		"/run/netns/cni-*",
	} {
		if err := unmountAndRemove(dir); err != nil {
			return err
		}
	}

	out, err := shx.Run(
		ctx,
		[]string{"ip", "netns", "show"},
		shx.CaptureOutput(),
		shx.PassStderr(),
		shx.WithCombinedError(),
	)
	if err != nil {
		return fmt.Errorf("failed to list network namespaces: %w", err)
	}
	for l, err := range iterLines(out.Stdout()) {
		if err != nil {
			return fmt.Errorf("error reading network namespaces: %w", err)
		}
		ns, _, _ := strings.Cut(l, " ")
		if err := multiSudo([]string{"remove netns " + ns, "ip", "netns", "delete", ns}); err != nil {
			return err
		}
	}
	out, err = shx.Run(
		ctx,
		[]string{"ip", "-o", "link", "show"},
		shx.CaptureOutput(),
		shx.PassStderr(),
		shx.WithCombinedError(),
	)
	if err != nil {
		return fmt.Errorf("failed to list network links: %w", err)
	}
	for l, err := range iterLines(out.Stdout()) {
		if err != nil {
			return fmt.Errorf("error reading network links: %w", err)
		}
		if !strings.Contains(l, " master cni0 ") {
			continue
		}
		fields := strings.Fields(l)
		if len(fields) < 2 {
			return fmt.Errorf("unexpected link line: %q", strings.TrimSpace(l))
		}
		link, _, _ := strings.Cut(fields[1], "@")
		if err := multiSudo([]string{"remove link " + link, "ip", "link", "delete", link}); err != nil {
			return err
		}
	}

	if err := multiSudo(
		[]string{"remove cni0", "ip", "link", "delete", "cni0"},
		[]string{"remove flannel.1", "ip", "link", "delete", "flannel.1"},
		[]string{"remove flannel-v6.1", "ip", "link", "delete", "flannel-v6.1"},
		[]string{"remove kube-ipvs0", "ip", "link", "delete", "kube-ipvs0"},
		[]string{"remove flannel-wg", "ip", "link", "delete", "flannel-wg"},
		[]string{"remove flannel-wg-v6", "ip", "link", "delete", "flannel-wg-v6"},
	); err != nil {
		return err
	}
	if err := unmountAndRemove("/var/lib/rancher/k3s"); err != nil {
		return err
	}

	if err := multiSudo(
		[]string{"remove /var/lib/cni", "rm", "-rf", "/var/lib/cni"},
		[]string{"remove /etc/rancher/k3s", "rm", "-rf", "/etc/rancher/k3s"},
		[]string{"remove /run/k3s", "rm", "-rf", "/run/k3s"},
		[]string{"remove /run/flannel", "rm", "-rf", "/run/flannel"},
		[]string{"remove /var/lib/kubelet", "rm", "-rf", "/var/lib/kubelet"},
		[]string{"remove /var/lib/rancher/k3s", "rm", "-rf", "/var/lib/rancher/k3s"},
	); err != nil {
		return err
	}

	if err := multiSudo(
		[]string{"remove k3s", "rm", "-f", addon.Config.k3sPath},
	); err != nil {
		return err
	}

	return nil
}

func iterLines(r io.Reader) iter.Seq2[string, error] {
	if r == nil {
		panic("iterLines called with nil reader")
	}
	s := bufio.NewScanner(r)
	return func(yield func(string, error) bool) {
		for s.Scan() {
			if !yield(s.Text(), nil) {
				return
			}
		}
		if err := s.Err(); err != nil {
			yield("", err)
		}
	}
}
