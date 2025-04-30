package k3s

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"fastcat.org/go/gdev/internal"
	"golang.org/x/sync/errgroup"
)

func killPods(ctx context.Context) error {
	// find all containerd-shim-ish processes whose exec is in
	// /var/lib/rancher/k3s/data and kill them and all their children.

	pids, err := shimPids(ctx)
	if err != nil {
		return err
	} else if len(pids) == 0 {
		fmt.Println("No pids to kill")
		return nil
	}
	pids, err = descendantPids(ctx, pids...)
	if err != nil {
		return err
	}
	fmt.Printf("Killing %d pids\n", len(pids))
	c := []string{"kill", "-9"}
	for _, pid := range pids {
		c = append(c, strconv.Itoa(pid))
	}
	if err := internal.Shell(
		ctx,
		c,
		internal.WithSudo("kill containerd-shim and children"),
	); err != nil {
		return err
	}

	return nil
}

func shimPids(ctx context.Context) ([]int, error) {
	const base = "/var/lib/rancher/k3s/data/"
	return procWalk(ctx, "cmdline", func(ctx context.Context, fn string) (int, error) {
		content, err := os.ReadFile(fn)
		if err != nil {
			return -1, err
		}
		commB, _, _ := bytes.Cut(content, []byte{0})
		comm := string(commB)
		if strings.HasPrefix(comm, base) && strings.HasPrefix(filepath.Base(comm), "containerd-shim") {
			pidStr := strings.TrimPrefix(fn, "/proc/")
			pidStr = strings.TrimSuffix(pidStr, "/cmdline")
			return strconv.Atoi(strings.Split(fn, "/")[2])
		}
		return 0, nil
	})
}

func descendantPids(ctx context.Context, pids ...int) ([]int, error) {
	ppids, err := procWalk(ctx, "status", func(ctx context.Context, fn string) ([2]int, error) {
		content, err := os.ReadFile(fn)
		if err != nil {
			return [2]int{}, err
		}
		pid, err := strconv.Atoi(strings.Split(fn, "/")[2])
		if err != nil {
			return [2]int{}, err
		}
		pfx := []byte("PPid:")
		for l := range bytes.Lines(content) {
			s, found := bytes.CutPrefix(l, pfx)
			if !found {
				continue
			}
			if ppid, err := strconv.Atoi(strings.TrimSpace(string(s))); err != nil {
				return [2]int{}, err
			} else {
				return [2]int{pid, ppid}, nil
			}
		}
		return [2]int{}, nil
	})
	if err != nil {
		return nil, err
	}
	children := make(map[int][]int, len(ppids)/2)
	for _, p := range ppids {
		children[p[1]] = append(children[p[1]], p[0])
	}
	ret := make([]int, 0, len(pids))
	ret = append(ret, pids...)
	q := slices.Clone(pids)
	for len(q) != 0 {
		// pop
		pid := q[len(q)-1]
		q = q[:len(q)-1]
		c := children[pid]
		ret = append(ret, c...)
		q = append(q, c...)
	}
	return ret, nil
}

func procWalk[T comparable](
	ctx context.Context,
	fn string,
	pf func(context.Context, string) (T, error),
) ([]T, error) {
	m, err := filepath.Glob("/proc/*/" + fn)
	if err != nil {
		// should be impossible
		return nil, err
	}
	fnCh := make(chan string)
	valCh := make(chan T)

	eg, ctx := errgroup.WithContext(ctx)
	wg, wgCtx := errgroup.WithContext(ctx)
	var vz T

	pfl := func() error {
		for fn := range fnCh {
			if val, err := pf(wgCtx, fn); err != nil {
				return err
			} else if val != vz {
				select {
				case <-wgCtx.Done():
					return nil
				case valCh <- val:
				}
			}
		}
		return nil
	}
	for range runtime.GOMAXPROCS(0) {
		wg.Go(pfl)
	}
	eg.Go(func() error {
		defer close(valCh)
		return wg.Wait()
	})
	var vals []T
	eg.Go(func() error {
		for val := range valCh {
			vals = append(vals, val)
		}
		return nil
	})
	eg.Go(func() error {
		defer close(fnCh)
		for _, fn := range m {
			if strings.HasPrefix(fn, "/proc/self") || strings.HasPrefix(fn, "/proc/thread-self") {
				continue
			}
			select {
			case <-ctx.Done():
				return nil
			case fnCh <- fn:
			}
		}
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return vals, nil
}
