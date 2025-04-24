package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"fastcat.org/go/gdev/pm/api"
	"github.com/stretchr/testify/assert"
)

func TestChildSleeps(t *testing.T) {
	if testing.Short() {
		t.SkipNow() // does not return
	}

	// run a simple sequence of init containers followed by a process container
	def := api.Child{
		Name: "sleeps",
		Init: []api.Exec{
			{
				Cmd:  "sleep",
				Args: []string{"0.1s"},
			},
			{
				Cmd:  "sleep",
				Args: []string{"0.2s"},
			},
		},
		Main: api.Exec{
			Cmd:  "sleep",
			Args: []string{"1h"}, // we will kill this one
		},
	}
	c := newChild(def)
	// TODO: this will hang if something goes wrong
	t.Cleanup(c.Wait)

	t.Log("initializing")
	go c.run()
	c.cmds <- childPing
	t.Log("starting")
	c.cmds <- childStart
	c.cmds <- childPing // sync
	for s := c.Status(); s.State != api.ChildRunning; s = c.Status() {
		t.Logf("child is %s", s.State)
		if !assert.Contains(t, []api.ChildState{api.ChildInitRunning, api.ChildRunning}, s.State) {
			// try to kill the child
			c.cmds <- childStop
			c.cmds <- childDelete
			t.FailNow()
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Logf("child is %s", c.Status().State)
	c.cmds <- childStop
	c.cmds <- childPing // sync
	for s := c.Status(); s.State != api.ChildStopped; s = c.Status() {
		t.Logf("child is %s", s.State)
		// we'd like to do ... something ... if this assert fails, but not clear
		// what we _can_ do
		assert.Equal(t, api.ChildStopping, s.State)
		time.Sleep(50 * time.Millisecond)
	}
	t.Logf("child is %s", c.Status().State)
	c.cmds <- childDelete
	c.Wait()
	s := c.Status()
	t.Logf("final: %#v", c.Status())
	assert.Equal(t, api.ChildStopped, s.State)
}

func TestChildFails(t *testing.T) {
	if testing.Short() {
		t.SkipNow() // does not return
	}

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
	c := newChild(def)
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
	assert.NoError(t, os.WriteFile(filepath.Join(td, "init1"), nil, 0644))
	// wait for it to restart and move on to the main process failing
	waitState(api.ChildError, api.ChildInitError, api.ChildInitRunning, api.ChildRunning)
	// let the main process move forwards
	assert.NoError(t, os.WriteFile(filepath.Join(td, "main"), nil, 0644))
	waitState(api.ChildRunning, api.ChildError)
	// TODO: make sure it stays there
	c.cmds <- childStop
	waitState(api.ChildStopped, api.ChildRunning)
	c.cmds <- childDelete
	c.Wait()
	s := c.Status()
	t.Logf("final: %#v", c.Status())
	assert.Equal(t, api.ChildStopped, s.State)
}
