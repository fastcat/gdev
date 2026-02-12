package sys

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"
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

func (s *systemdIsolator) Isolate(
	ctx context.Context,
	name string,
	process *os.Process,
) (string, error) {
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
			dbus.PropPids(uint32(process.Pid)),
			// accounting copied from containerd/cgroups/v3/cgroups2
			{Name: "MemoryAccounting", Value: godbus.MakeVariant(true)},
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
		return name, fmt.Errorf("service isolation for %s (%d) failed: %s", name, process.Pid, status)
	}
}

func (s *systemdIsolator) Cleanup(ctx context.Context, group string) error {
	conn, err := s.getConn()
	if err != nil {
		return err
	}
	ch := make(chan string, 1)
	_, err = conn.StopUnitContext(ctx, group, "fail", ch)
	if err != nil {
		// suppress a common error where systemd has already removed the empty scope
		// on exit
		if err.Error() == "Unit "+group+" not loaded." {
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
		return fmt.Errorf("service termination for %s failed: %s", group, status)
	}
}

type cgroupsIsolator struct {
	parentGroup string
}

func (c *cgroupsIsolator) Isolate(
	ctx context.Context,
	name string,
	process *os.Process,
) (string, error) {
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
	if err := mgr.AddProc(uint64(process.Pid)); err != nil {
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

func (c *cgroupsIsolator) getParentGroup() (string, error) {
	if c.parentGroup == "" {
		// default: put the new scope below whatever contains the current process
		cur, err := cgroup2.PidGroupPath(os.Getpid())
		if err != nil {
			return cur, err
		}
		if err := c.tryParentGroup(cur); err != nil {
			return cur, err
		}
	}
	return c.parentGroup, nil
}

func (c *cgroupsIsolator) tryParentGroup(cur string) error {
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
