package server

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fastcat.org/go/gdev/addons/pm/api"
	"fastcat.org/go/gdev/lib/sys"
)

func TestChildSleeps(t *testing.T) {
	if testing.Short() {
		t.SkipNow() // does not return
	}

	isolator, err := sys.GetIsolator()
	require.NoError(t, err)

	// run a simple sequence of init containers followed by a process container
	def := api.Child{
		Name: "sleeps",
		Init: []api.Exec{
			{
				Cmd:  "sleep",
				Args: []string{"0.05s"},
			},
			{
				Cmd:  "sleep",
				Args: []string{"0.1s"},
			},
		},
		Main: api.Exec{
			Cmd:  "sleep",
			Args: []string{"1h"}, // we will kill this one
		},
	}
	c := newChild(def, isolator)
	// TODO: this will hang if something goes wrong
	t.Cleanup(c.Wait)

	t.Log("initializing")
	runChild(t, c, 25*time.Millisecond)
}

func TestChildFails(t *testing.T) {
	if testing.Short() {
		t.SkipNow() // does not return
	}

	isolator, err := sys.GetIsolator()
	require.NoError(t, err)

	td := t.TempDir()

	// run a simple sequence of init containers followed by a process container
	def := api.Child{
		Name: "sleeps",
		Init: []api.Exec{
			{
				Cmd:  "test",
				Args: []string{"-f", "init1"},
				Cwd:  td,
			},
		},
		Main: api.Exec{
			Cmd: "/bin/sh",
			// sleep if the test succeeds so we don't get child error and can kill it
			// off
			Args: []string{"-c", "test -f main && sleep 1h"},
			Cwd:  td,
		},
	}
	c := newChild(def, isolator)
	// speed up the restart timers to make this test not so slow
	c.restartDelay = 20 * time.Millisecond

	// TODO: this will hang if something goes wrong
	t.Cleanup(c.Wait)

	t.Log("initializing")
	go c.run()
	c.cmds <- childPing
	t.Log("starting")
	c.cmds <- childStart
	c.cmds <- childPing // sync

	waitState := func(want api.ChildState, allowed ...api.ChildState) {
		t.Logf("waiting for child to be %s", want)
		var last api.ChildState
		for s := c.Status(); s.State != want; s = c.Status() {
			if s.State != last {
				t.Logf("child is %s", s.State)
				last = s.State
			}
			if len(allowed) > 0 {
				if !assert.Contains(t, allowed, s.State) {
					// try to kill it off
					c.cmds <- childStop
					c.cmds <- childDelete
					t.FailNow()
				}
			}
			time.Sleep(5 * time.Millisecond)
		}
	}

	// we expect the init process to fail, wait for that and then "touch" the file
	// that will allow it to succeed
	waitState(api.ChildInitError, api.ChildInitRunning)
	require.NoError(t, os.WriteFile(filepath.Join(td, "init1"), nil, 0o644))
	// wait for it to restart and move on to the main process failing
	waitState(api.ChildError, api.ChildInitError, api.ChildInitRunning, api.ChildRunning)
	// let the main process move forwards
	require.NoError(t, os.WriteFile(filepath.Join(td, "main"), nil, 0o644))
	waitState(api.ChildRunning, api.ChildError)
	// TODO: make sure it stays there
	c.cmds <- childStop
	waitState(api.ChildStopped, api.ChildStopping, api.ChildRunning)
	c.cmds <- childDelete
	c.Wait()
	s := c.Status()
	t.Logf("final: %#v", c.Status())
	assert.Equal(t, api.ChildStopped, s.State)
}

func TestChildLogs(t *testing.T) {
	isolator, err := sys.GetIsolator()
	require.NoError(t, err)
	td := t.TempDir()
	def := api.Child{
		Name: "test1",
		Main: api.Exec{
			Cmd:     "sh",
			Args:    []string{"-c", "echo hello ; echo world 1>&2"},
			Logfile: filepath.Join(td, "test1-main.log"),
		},
		Init: []api.Exec{
			{
				Cmd:     "sh",
				Args:    []string{"-c", "echo init1out ; echo init1err 1>&2"},
				Logfile: filepath.Join(td, "test1-init1.log"),
			},
		},
		// we need to use one-shot mode here because otherwise the children exit too
		// fast and we see them failing
		OneShot: true,
	}
	c := newChild(def, isolator)
	t.Cleanup(c.Wait)
	if !runChild(t, c, time.Millisecond) {
		return
	}

	// check the logs we wrote
	initLog1, err := os.ReadFile(filepath.Join(td, "test1-init1.log"))
	require.NoError(t, err)
	assert.Equal(t, "init1out\ninit1err\n", string(initLog1))

	mainLog, err := os.ReadFile(filepath.Join(td, "test1-main.log"))
	require.NoError(t, err)
	assert.Equal(t, "hello\nworld\n", string(mainLog))
}

func runChild(t *testing.T, c *child, pollRate time.Duration) bool {
	success := true
	go c.run()
	c.cmds <- childPing
	c.cmds <- childStart
	c.cmds <- childPing
	startedStates := []api.ChildState{api.ChildRunning}
	startingStates := append([]api.ChildState{api.ChildInitRunning}, startedStates...)
	stoppedStates := []api.ChildState{api.ChildStopped}
	if c.def.OneShot {
		startedStates = append(startedStates, api.ChildDone)
		startingStates = append(startingStates, api.ChildDone)
		stoppedStates = append(stoppedStates, api.ChildDone)
	}
	for s := c.Status(); !slices.Contains(startedStates, s.State); s = c.Status() {
		t.Logf("child is %s", s.State)
		if !assert.Contains(t, startingStates, s.State) {
			// try to kill the child
			c.cmds <- childStop
			c.cmds <- childDelete
			success = false // doesn't matter because FailNow
			t.FailNow()     // does not return
		}
		time.Sleep(pollRate)
	}
	t.Logf("child is %s", c.Status().State)
	// don't stop one-shots, wait for them to exit instead
	if c.def.OneShot {
		for s := c.Status(); s.State != api.ChildDone; s = c.Status() {
			t.Logf("child is %s", s.State)
			if !assert.Contains(t, startingStates, s.State) {
				// try to kill the child
				c.cmds <- childStop
				c.cmds <- childDelete
				success = false // doesn't matter because FailNow
				t.FailNow()     // does not return
			}
			time.Sleep(pollRate)
		}
		t.Logf("child is %s", c.Status().State)
	} else {
		c.cmds <- childStop
	}
	c.cmds <- childPing // sync
	for s := c.Status(); !slices.Contains(stoppedStates, s.State); s = c.Status() {
		t.Logf("child is %s", s.State)
		// we'd like to do ... something ... if this assert fails, but not clear
		// what we _can_ do
		success = assert.Equal(t, api.ChildStopping, s.State) && success
		time.Sleep(pollRate)
	}
	t.Logf("child is %s", c.Status().State)
	c.cmds <- childDelete
	c.Wait()
	s := c.Status()
	t.Logf("final: %#v", c.Status())
	if c.def.OneShot {
		success = assert.Equal(t, api.ChildDone, s.State) && success
	} else {
		success = assert.Equal(t, api.ChildStopped, s.State) && success
	}
	return success
}
