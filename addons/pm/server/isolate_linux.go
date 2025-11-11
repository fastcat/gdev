package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"

	"fastcat.org/go/gdev/instance"
)

func init() {
	getIsolator = sync.OnceValues(func() (isolator, error) {
		if canSystemd(context.Background()) {
			return &systemdIsolator{}, nil
		} else {
			return &cgroupsIsolator{}, nil
		}
	})
}

func canSystemd(ctx context.Context) bool {
	conn, err := dbus.NewUserConnectionContext(ctx)
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
	conn, err := dbus.NewUserConnectionContext(context.Background())
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

func (s *systemdIsolator) isolateProcess(
	ctx context.Context,
	name string,
	process *os.Process,
) (string, error) {
	conn, err := s.getConn()
	if err != nil {
		return "", err
	}
	ch := make(chan string, 1)
	unitName := instance.AppName() + "-pm-" + name + ".scope"
	_, err = conn.StartTransientUnitContext(
		ctx,
		unitName,
		"fail", // error if unit already exists
		[]dbus.Property{
			dbus.PropDescription(fmt.Sprintf("%s pm service %s", instance.AppName(), name)),
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
			return unitName, nil
		}
		return unitName, fmt.Errorf("service isolation for %s (%d) failed: %s", name, process.Pid, status)
	}
}

func (s *systemdIsolator) cleanup(ctx context.Context, group string) error {
	conn, err := s.getConn()
	if err != nil {
		return err
	}
	ch := make(chan string, 1)
	_, err = conn.StopUnitContext(ctx, group, "fail", ch)
	if err != nil {
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

func (c *cgroupsIsolator) isolateProcess(
	ctx context.Context,
	name string,
	process *os.Process,
) (string, error) {
	cur := c.parentGroup
	if c.parentGroup == "" {
		// default: put the new scope below whatever contains the current process
		var err error
		cur, err = cgroup2.PidGroupPath(os.Getpid())
		if err != nil {
			return "", err
		}
	}
	groupPath := filepath.Join(cur, instance.AppName()+"-pm-"+name+".scope")
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

func (*cgroupsIsolator) cleanup(ctx context.Context, groupPath string) error {
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

const cgroupsMountPath = "/sys/fs/cgroup"
