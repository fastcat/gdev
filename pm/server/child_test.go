package server

import (
	"testing"
	"time"

	"fastcat.org/go/gdev/pm/api"
	"github.com/stretchr/testify/assert"
)

func TestChildSlow(t *testing.T) {
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
