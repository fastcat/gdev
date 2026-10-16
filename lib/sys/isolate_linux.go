package sys

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"

	"fastcat.org/go/gdev/instance"
)

func init() {
	GetIsolator = sync.OnceValues(func() (Isolator, error) {
		if canSystemd(context.Background()) {
			return &systemdIsolator{}, nil
		} else {
			i := &cgroupsIsolator{}
			if _, err := i.getParentGroup(); err != nil {
				return nil, err
			}
			return i, nil
		}
	})
}

func canSystemd(ctx context.Context) bool {
	conn, err := SystemdUserConn(ctx)
	if err != nil {
		return false
	}
	defer conn.Close() //nolint:errcheck

	// make sure systemd is running, not just dbus
	if _, err := conn.SystemStateContext(ctx); err != nil {
		return false
	}

	return true
}

type systemdIsolator struct {
	conn atomic.Pointer[dbus.Conn]
}

func (s *systemdIsolator) getConn() (*dbus.Conn, error) {
	conn := s.conn.Load()
	if conn != nil {
		return conn, nil
	}
	conn, err := SystemdUserConn(context.Background())
	if err != nil {
		return nil, err
	}
	if !s.conn.CompareAndSwap(nil, conn) {
		// lost the race, close the excess connection
		conn.Close() //nolint:errcheck
		conn = s.conn.Load()
	}
	// we never close the retained connection, it stays open for reuse for the
	// life of the process
	return conn, nil
}

func (s *systemdIsolator) Isolate(ctx context.Context, name string, pid int) (string, error) {
	// systemd won't allow moving an existing pid into a .service
	if !strings.HasSuffix(name, ".scope") {
		return "", fmt.Errorf("unit name %q must end with .scope", name)
	}

	conn, err := s.getConn()
	if err != nil {
		return "", err
	}
	ch := make(chan string, 1)
	_, err = conn.StartTransientUnitContext(
		ctx,
		name,
		"fail", // error if unit already exists
		[]dbus.Property{
			// TODO: description is contextual and needs to be passed in not derived
			// dbus.PropDescription(fmt.Sprintf("%s pm service %s", instance.AppName(), name)),

			// auto-harvest the transient unit once all its processes exit
			{Name: "CollectMode", Value: godbus.MakeVariant("inactive-or-failed")},
			// put the given PID into it now
			dbus.PropPids(uint32(pid)),
			// accounting copied from containerd/cgroups/v3/cgroups2
			{Name: "MemoryAccounting", Value: godbus.MakeVariant(true)},
			// NOTE: always enabled in newer systemd, this is a failsafe for older systems
			{Name: "CPUAccounting", Value: godbus.MakeVariant(true)},
			{Name: "IOAccounting", Value: godbus.MakeVariant(true)},
		},
		ch,
	)
	if err != nil {
		return "", err
	}
	select {
	case <-ctx.Done():
		// TODO: what to do about the dangling systemd job?
		return "", ctx.Err()
	case status := <-ch:
		if status == "done" {
			return name, nil
		}
		// NOTE: this is a pretty common error if the process exits fast
		return name, fmt.Errorf("service isolation for %s (%d) failed: %s", name, pid, status)
	}
}

func (s *systemdIsolator) Cleanup(ctx context.Context, unit string) error {
	conn, err := s.getConn()
	if err != nil {
		return err
	}
	ch := make(chan string, 1)
	_, err = conn.StopUnitContext(ctx, unit, "fail", ch)
	if err != nil {
		// suppress a common error where systemd has already removed the empty scope
		// on exit
		if err.Error() == "Unit "+unit+" not loaded." {
			return nil
		}
		return err
	}
	select {
	case <-ctx.Done():
		// TODO: what to do about the dangling systemd job?
		return ctx.Err()
	case status := <-ch:
		if status == "done" {
			return nil
		}
		return fmt.Errorf("service termination for %s failed: %s", unit, status)
	}
}

func (s *systemdIsolator) EnableUsage(ctx context.Context) error {
	// systemd automatically enables memory control, and CPU is always available,
	// so this is a no-op
	return nil
}

func (s *systemdIsolator) Usage(ctx context.Context, unit string) (IsolateUsage, error) {
	group, err := s.resolveCgroup(ctx, unit)
	if err != nil {
		return IsolateUsage{}, err
	}
	return cgroupUsage(group)
}

func (s *systemdIsolator) resolveCgroup(ctx context.Context, unit string) (string, error) {
	conn, err := s.getConn()
	if err != nil {
		return "", err
	}
	// look up the cgroup path for the unit
	p, err := conn.GetUnitTypePropertyContext(ctx, unit, "Scope", "ControlGroup")
	if err != nil {
		return "", err
	}
	cgPath, ok := p.Value.Value().(string)
	if !ok {
		return "", fmt.Errorf("unexpected type for ControlGroup property: %T", p.Value.Value())
	} else if cgPath == "" {
		return "", fmt.Errorf("empty ControlGroup property for unit %q", unit)
	}
	return cgPath, nil
}

type cgroupsIsolator struct {
	parentGroup string
}

func (c *cgroupsIsolator) Isolate(ctx context.Context, name string, pid int) (string, error) {
	// don't apply the same rules as systemd so we can fake a .service we would
	// have started via it
	if !strings.HasSuffix(name, ".scope") && !strings.HasSuffix(name, ".service") {
		return "", fmt.Errorf("unit name %q must end with .scope or .service", name)
	}
	cur, err := c.getParentGroup()
	if err != nil {
		return "", err
	}
	groupPath := filepath.Join(cur, name)
	mgr, err := cgroup2.NewManager(
		cgroupsMountPath,
		// TODO: hierarchy?
		groupPath,
		&cgroup2.Resources{},
	)
	if err != nil {
		return groupPath, err
	}
	if err := mgr.AddProc(uint64(pid)); err != nil {
		return groupPath, err
	}
	// TODO: need to delete the cgroup when the process exits
	return groupPath, nil
}

func (*cgroupsIsolator) Cleanup(ctx context.Context, groupPath string) error {
	mgr, err := cgroup2.Load(groupPath)
	if err != nil {
		return err
	}
	// the cgroups library produces yucky console warnings if we try to clean
	// up a cgroup that is already gone (e.g. the child has already exited). We
	// do a read (what we read doesn't really matter) to produce the ErrNotExist
	// to detect this case and skip the cleanup logic.
	if _, err := mgr.GetType(); err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err := mgr.Kill(); err != nil {
		return err
	}
	if err := mgr.Delete(); err == nil {
		return nil
	}
	// give it a moment to harvest dead processes
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	retry := time.NewTicker(5 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return mgr.Delete()
		case <-retry.C:
			if err := mgr.Delete(); err == nil {
				return nil
			}
		}
	}
}

func (c *cgroupsIsolator) EnableUsage(ctx context.Context) error {
	pg, err := c.getParentGroup()
	if err != nil {
		return err
	}
	mgr, err := cgroup2.Load(pg)
	if err != nil {
		return err
	}
	controllers, err := mgr.Controllers()
	if err != nil {
		return err
	}
	if !slices.Contains(controllers, "memory") {
		return fmt.Errorf("memory controller not available to delegate in %s", pg)
	}

	// the root cgroup has special rules, but if we're in the container, what
	// looks like the root may not really be so.
	if _, err := mgr.GetType(); errors.Is(err, os.ErrNotExist) {
		// this is the real root cgroup, no action is required
		return nil
	}

	// in order to enable subtree control we have to evict all PIDs from the
	// parent group into a sub-group. However we don't want to do that if we
	// already did it, so check if we seem to be using one of those, and if so
	// shift up to using its parent. Since we normally look for the _parent_ group
	// of the current process, we generally shouldn't hit this branch, it's just
	// belt & suspenders.
	altRootName := instance.AppName() + "-isolate-root.scope"
	if filepath.Base(pg) == altRootName {
		// we were already moved into one of these alt roots. all we need to do is
		// use its parent
		c.parentGroup = filepath.Dir(pg)
		return nil
	}

	altRoot, err := mgr.NewChild(altRootName, nil)
	if err != nil {
		return fmt.Errorf("failed to create cgroup to move away procs: %w", err)
	}
	for {
		pids, err := mgr.Procs(false)
		if err != nil {
			return fmt.Errorf("failed to list pids in group %s: %w", pg, err)
		} else if len(pids) == 0 {
			break
		}
		for _, pid := range pids {
			if err := altRoot.AddProc(pid); err != nil {
				return fmt.Errorf("failed to move pid %d to alternate root cgroup: %w", pid, err)
			}
		}
	}
	// enabling memory control in the sub-group means enabling delegation for
	// everything up to but not including it.
	if err := altRoot.ToggleControllers([]string{"memory"}, cgroup2.Enable); err != nil {
		return fmt.Errorf("unable to delegate memory control: %w", err)
	}
	return nil
}

func cgroupUsage(group string) (IsolateUsage, error) {
	mgr, err := cgroup2.Load(group)
	if err != nil {
		return IsolateUsage{}, err
	}
	m, err := mgr.StatFiltered(cgroup2.StatCPU | cgroup2.StatMemory)
	if err != nil {
		return IsolateUsage{}, err
	}
	return IsolateUsage{
		User:       time.Duration(m.CPU.UserUsec) * time.Microsecond,
		System:     time.Duration(m.CPU.SystemUsec) * time.Microsecond,
		Memory:     m.Memory.Usage,
		MemoryPeak: m.Memory.MaxUsage,
	}, nil
}

func (*cgroupsIsolator) Usage(ctx context.Context, group string) (IsolateUsage, error) {
	return cgroupUsage(group)
}

func (c *cgroupsIsolator) getParentGroup() (string, error) {
	if c.parentGroup != "" {
		return c.parentGroup, nil
	}
	// default: put the new scope below whatever contains the current process
	cur, err := cgroup2.PidGroupPath(os.Getpid())
	if err != nil {
		return cur, err
	}
	// we want to be able to do usage tracking in isolated children, so we
	// really want to create things in the parent of our own cgroup. But if we
	// are in the root cgroup, there's nowhere up to go, but Dir will just
	// return the root path back in that case anyways.
	par := filepath.Dir(cur)
	candidates := []string{par, cur}
	if par == cur {
		candidates = candidates[1:]
	}
	var errs []error
	for _, candidate := range candidates {
		if err := c.tryGroup(candidate); err != nil {
			errs = append(errs, err)
		} else {
			c.parentGroup = candidate
			return c.parentGroup, nil
		}
	}
	return "", errors.Join(errs...)
}

func (c *cgroupsIsolator) tryGroup(cur string) error {
	// if we aren't root, we need to check permissions to see if we can
	// manipulate our own cgroup
	if uid, gid := os.Geteuid(), os.Getegid(); uid != 0 {
		st, err := os.Stat(filepath.Join(cgroupsMountPath, cur))
		if err != nil {
			return err
		}
		if stt, ok := st.Sys().(*syscall.Stat_t); ok {
			if stt.Uid != uint32(uid) &&
				(stt.Gid != uint32(gid) || (stt.Mode&syscall.S_IWGRP) == 0) &&
				(stt.Mode&syscall.S_IWOTH) == 0 {
				// we probably can't create sub-groups here (unless there are ACLs)
				return fmt.Errorf("insufficient permissions to create cgroup in %q: %w", cur, &os.PathError{
					Op:   "cgroup",
					Path: cur,
					Err:  os.ErrPermission,
				})
			}
		}
	}

	// TODO: harmless? data race depending on usage
	c.parentGroup = cur

	return nil
}

const cgroupsMountPath = "/sys/fs/cgroup"
