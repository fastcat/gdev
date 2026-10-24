package sys

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/containerd/log"
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

		unit, err := i.Isolate(t.Context(), "test-sleep.scope", cmd.Process.Pid)
		require.NoError(t, err)
		t.Logf("started unit %q", unit)
		g, err := i.resolveCgroup(t.Context(), unit)
		require.NoError(t, err)
		g2, err := cgroup2.PidGroupPath(cmd.Process.Pid)
		require.NoError(t, err)
		t.Logf("systemd says unit %q is in cgroup %q", unit, g)
		assert.Equal(t, g2, g, "systemd cgroup lookup should match native pid-based lookup")
		usage, err := i.Usage(t.Context(), unit)
		if assert.NoError(t, err) {
			t.Logf("cpu usage is: %+#v", usage)
		}
		mgr, err := cgroup2.Load(g)
		require.NoError(t, err)
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

		unit, err := i.Isolate(t.Context(), "test-sleep.scope", cmd.Process.Pid)
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
		group, err := i.Isolate(t.Context(), "test-sleep.scope", cmd.Process.Pid)
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
		group, err := i.Isolate(t.Context(), "test-double-sleep.service", cmd.Process.Pid)
		require.NoError(t, err)
		t.Logf("started first process in cgroup %q", group)
		require.NoError(t, cmd.Process.Kill())
		_, err = cmd.Process.Wait()
		require.NoError(t, err)

		cmd = startSleep(t)
		group2, err := i.Isolate(t.Context(), "test-double-sleep.service", cmd.Process.Pid)
		require.NoError(t, err)
		t.Logf("started second process in cgroup %q", group2)
		assert.Equal(t, group, group2)
		require.NoError(t, i.Cleanup(t.Context(), group2))
	})
}

func Test_cgroupsIsolator_Cleanup_Noise(t *testing.T) {
	i := cgroupsIsolator{}
	// see notes above for why this might fail
	if _, err := i.getParentGroup(); err != nil {
		t.SkipNow()
	}

	// spy on the logger the containerd lib uses
	origLogger := log.L.Logger
	t.Cleanup(func() { log.L.Logger = origLogger })
	var buf bytes.Buffer
	log.L.Logger.SetOutput(&buf)

	cmd := startSleep(t)
	group, err := i.Isolate(t.Context(), "test-sleep.scope", cmd.Process.Pid)
	require.NoError(t, err)

	assert.NoError(t, i.Cleanup(t.Context(), group))

	// cleanup again and make sure it doesn't log errors
	assert.NoError(t, i.Cleanup(t.Context(), group))

	assert.Empty(t, buf.String(), "double cleanup should not log")
}

func Test_cgroupsIsolator_EnableUsage(t *testing.T) {
	// we need to create an alternate cgroup and make the isolator think that it
	// is its own
	i := &cgroupsIsolator{}
	_, err := i.getParentGroup()
	if errors.Is(err, os.ErrPermission) {
		t.Skipf("cgroups access denied: %v", err)
	}
	require.NoError(t, err)
	// If we are in the root cgroup, whether or not it is the true one, then we
	// can't run this test without semi-permanently changing it, which we should
	// not do.
	if i.parentGroup == "/" {
		t.Skip("cannot run test when in the root cgroup")
	}
	// whatever other cgroup this process is in, it's there, and so we won't be able to
	// enable subtree delegation unless we are in the root cgroup. If we aren't in
	// the root, try going up a level, hopefully that will be a systemd slice we
	// can use for testing.
	i.parentGroup = filepath.Dir(i.parentGroup)

	// create the child group and point the isolator at it
	pgm, err := cgroup2.Load(i.parentGroup)
	require.NoError(t, err)

	cgm, err := pgm.NewChild("test.scope", nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, cgm.Delete())
	})

	// start a new process in the test scope
	cmd := startSleep(t)
	t.Logf("process %d started", cmd.Process.Pid)
	group, err := i.Isolate(t.Context(), "test.scope", cmd.Process.Pid)
	require.NoError(t, err)
	t.Logf("process %d isolated into %q", cmd.Process.Pid, group)

	// now we actually do the test: make another isolator with the test scope as
	// its root, and verify that it can move the sleep process into another
	// sub-group and enable memory control delegation on the test scope.
	ti := &cgroupsIsolator{filepath.Join(i.parentGroup, "test.scope")}
	require.NoError(t, ti.EnableUsage(t.Context()),
		"should be able to enable memory delegation by moving away a pid",
	)

	// now the child process should be in a different cgroup
	movedGroup, err := cgroup2.PidGroupPath(cmd.Process.Pid)
	require.NoError(t, err)
	assert.NotEqual(t, group, movedGroup,
		"the process should have been moved to a different cgroup",
	)
	t.Logf("process %d moved to %q", cmd.Process.Pid, movedGroup)
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
	i := &cgroupsIsolator{}
	_, err := i.getParentGroup()
	if err == nil {
		// we could set this up, but it would interfere with all the other tests, so
		// we would need to make this test re-exec and run in a child process
		t.Skipf("this test requires current process to be in cgroup it can't control")
	}
	// this needs to be a group with no subtree control so that we can add processes into it
	err = i.tryGroup(fmt.Sprintf("/user.slice/user-%[1]d.slice/user@%[1]d.service/init.scope", os.Geteuid()))
	if err != nil {
		t.Skipf("need the systemd user service cgroup to be available for test: %v", err)
	}

	// of course, if we don't control our own cgroup, we can't fork a child into
	// some other cgroup, because that logically requires permission to remove
	// things from our current group ... which we don't control.
	t.Skip("cgroups don't work this way")

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
