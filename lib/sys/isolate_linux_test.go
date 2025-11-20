package sys

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_systemdIsolator_isolateProcess(t *testing.T) {
	if !canSystemd(t.Context()) {
		t.Skip("systemd user instance unavailable")
	}
	i := &systemdIsolator{}
	conn, err := i.getConn()
	// if canSystemd said yes, this should not fail, since they are doing the same
	// thing
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	t.Run("kill via cgroup", func(t *testing.T) {
		cmd := startSleep(t)

		unit, err := i.Isolate(t.Context(), "test-sleep.scope", cmd.Process)
		require.NoError(t, err)
		t.Logf("started unit %q", unit)
		g, err := cgroup2.PidGroupPath(cmd.Process.Pid)
		require.NoError(t, err)
		mgr, err := cgroup2.Load(g)
		require.NoError(t, err)
		t.Logf("found unit in cgroup %q", g)
		require.NoError(t, mgr.Kill())
		ps, err := cmd.Process.Wait()
		if assert.NoError(t, err) {
			// NOTE: ps.Exited() returns false in this case, and thus
			// ps.ExitCode() returns -1, not -128+signal like one would expect in
			// a shell environment.
			ws := ps.Sys().(syscall.WaitStatus)
			assert.True(t, ws.Signaled())
			assert.Equal(t, syscall.SIGKILL, ws.Signal())
		}
		// wait for up to 1 second for the cgroup to be removed by systemd
		deadline, cancel := context.WithTimeout(t.Context(), time.Second)
		t.Cleanup(cancel)
		retry := time.NewTicker(10 * time.Millisecond)
		defer retry.Stop()
		done := false
		for !done {
			select {
			case <-deadline.Done():
				done = true
			case <-retry.C:
				//cspell:ignore Procs
				_, err = mgr.Procs(false)
				if err != nil || done {
					if assert.ErrorIs(t, err, os.ErrNotExist) {
						done = true
					}
				}
			}
		}
	})
	t.Run("kill via systemd", func(t *testing.T) {
		cmd := startSleep(t)

		unit, err := i.Isolate(t.Context(), "test-sleep.scope", cmd.Process)
		require.NoError(t, err)
		t.Logf("started unit %q", unit)
		require.NoError(t, i.Cleanup(t.Context(), unit))
		// TODO: assert this takes roughly 0 time because the process is already exited
		ps, err := cmd.Process.Wait()
		if assert.NoError(t, err) {
			// NOTE: ps.Exited() returns false in this case, and thus
			// ps.ExitCode() returns -1, not -128+signal like one would expect in
			// a shell environment.
			ws := ps.Sys().(syscall.WaitStatus)
			assert.True(t, ws.Signaled())
			// systemd starts with SIGTERM, we expect that to have worked
			assert.Equal(t, syscall.SIGTERM, ws.Signal())
		}
		// TODO: assert that the cgroup is removed, that's mostly just asserting
		// that we configured systemd correctly and it did its job, not worth
		// testing here.
	})
}

func Test_cgroupsIsolator_isolateProcess(t *testing.T) {
	i := cgroupsIsolator{}
	cur, err := i.getParentGroup()
	t.Logf("current cgroup: %q (ok? %v)", cur, err == nil)
	// if our current cgroup is not writable, finding another cgroup to move
	// processes into won't help, because we need permissions on our cgroup where
	// the PID begins in order to move it. Otherwise we could try
	// /user.slice/user-<uid>.slice/user@<uid>.service.
	if err != nil {
		t.SkipNow()
	}

	t.Run("create and cleanup", func(t *testing.T) {
		cmd := startSleep(t)
		group, err := i.Isolate(t.Context(), "test-sleep.scope", cmd.Process)
		require.NoError(t, err)
		t.Logf("started process in cgroup %q", group)
		g, err := cgroup2.PidGroupPath(cmd.Process.Pid)
		require.NoError(t, err)
		assert.Equal(t, group, g)
		assert.Equal(t, "test-sleep.scope", filepath.Base(g))
		assert.NoError(t, i.Cleanup(t.Context(), group))
	})
	t.Run("double create", func(t *testing.T) {
		// this test is to verify we can recover from an unclean shutdown that left
		// the empty cgroup behind
		cmd := startSleep(t)
		group, err := i.Isolate(t.Context(), "test-double-sleep.service", cmd.Process)
		require.NoError(t, err)
		t.Logf("started first process in cgroup %q", group)
		require.NoError(t, cmd.Process.Kill())
		_, err = cmd.Process.Wait()
		require.NoError(t, err)

		cmd = startSleep(t)
		group2, err := i.Isolate(t.Context(), "test-double-sleep.service", cmd.Process)
		require.NoError(t, err)
		t.Logf("started second process in cgroup %q", group2)
		assert.Equal(t, group, group2)
		require.NoError(t, i.Cleanup(t.Context(), group2))
	})
}

func startSleep(t *testing.T) *exec.Cmd {
	cmd := exec.CommandContext(t.Context(), "sleep", "1h")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		cmd.Process.Kill() //nolint:errcheck
		cmd.Process.Wait() //nolint:errcheck
	})
	return cmd
}

func TestTryCloneIntoGroup(t *testing.T) {
	t.Skipf("this test shows how to use CLONE_INTO_CGROUP, but it does not solve the permissions issues")
	i := &cgroupsIsolator{}
	_, err := i.getParentGroup()
	if err == nil {
		t.Skipf("this test requires current process to be in cgroup it can't control")
	}
	err = i.tryParentGroup(fmt.Sprintf("/user.slice/user-%[1]d.slice/user@%[1]d.service", os.Geteuid()))
	if err != nil {
		t.Skipf("need the systemd user service cgroup to be available for test: %v", err)
	}
	cgd := filepath.Join(cgroupsMountPath, i.parentGroup)
	cgf, err := os.Open(cgd)
	require.NoError(t, err)
	defer cgf.Close() //nolint:errcheck
	cmd := exec.CommandContext(t.Context(), "sleep", "0s")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		UseCgroupFD: true,
		CgroupFD:    int(cgf.Fd()),
	}
	require.NoError(t, cmd.Run())
}
