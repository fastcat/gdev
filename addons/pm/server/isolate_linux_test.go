package server

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fastcat.org/go/gdev/instance"
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

		unit, err := i.isolateProcess(t.Context(), "test-sleep", cmd.Process)
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

		unit, err := i.isolateProcess(t.Context(), "test-sleep", cmd.Process)
		require.NoError(t, err)
		t.Logf("started unit %q", unit)
		require.NoError(t, i.cleanup(t.Context(), unit))
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
	cur, err := cgroup2.PidGroupPath(os.Getpid())
	require.NoError(t, err)
	t.Logf("current cgroup: %q", cur)
	if strings.Contains(cur, "system.slice") {
		// happens in CI, we can't attach child cgroups here, and the user slice we
		// can create cgroups but can't move processes into them
		t.Skip("running in system.slice, cannot manage cgroups here")
	}

	t.Run("create and cleanup", func(t *testing.T) {
		cmd := startSleep(t)
		group, err := i.isolateProcess(t.Context(), "test-sleep", cmd.Process)
		require.NoError(t, err)
		t.Logf("started process in cgroup %q", group)
		g, err := cgroup2.PidGroupPath(cmd.Process.Pid)
		require.NoError(t, err)
		assert.Equal(t, group, g)
		assert.Equal(t, instance.AppName()+"-pm-test-sleep.scope", filepath.Base(g))
		assert.NoError(t, i.cleanup(t.Context(), group))
	})
	t.Run("double create", func(t *testing.T) {
		// this test is to verify we can recover from an unclean shutdown that left
		// the empty cgroup behind
		cmd := startSleep(t)
		group, err := i.isolateProcess(t.Context(), "test-double-sleep", cmd.Process)
		require.NoError(t, err)
		t.Logf("started first process in cgroup %q", group)
		require.NoError(t, cmd.Process.Kill())
		_, err = cmd.Process.Wait()
		require.NoError(t, err)

		cmd = startSleep(t)
		group2, err := i.isolateProcess(t.Context(), "test-double-sleep", cmd.Process)
		require.NoError(t, err)
		t.Logf("started second process in cgroup %q", group2)
		assert.Equal(t, group, group2)
		require.NoError(t, i.cleanup(t.Context(), group2))
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
