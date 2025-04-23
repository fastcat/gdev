package server

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"slices"
	"sync/atomic"
	"syscall"
	"time"

	"fastcat.org/go/gdev/pm/api"
)

type child struct {
	def    api.Child
	status atomic.Pointer[api.ChildStatus]
	cmds   chan childCmd
}

func newChild(def api.Child) *child {
	c := &child{
		def:  def,
		cmds: make(chan childCmd), // important that this be un-buffered
	}
	s := initialStatus(c)
	c.status.Store(&s)
	return c
}

type childCmd string

const (
	childPing   childCmd = "ping"
	childStart  childCmd = "start"
	childStop   childCmd = "stop"
	childDelete childCmd = "delete"
)

func (c *child) run() {
	status := initialStatus(c)
	c.status.Store(cloneStatus(status))

	curExec := -1 // initially nothing is running
	curStatus := func() *api.ExecStatus {
		if curExec < 0 {
			return nil
		} else if curExec < len(status.Init) {
			return &status.Init[curExec]
		} else {
			return &status.Main
		}
	}
	var curProc *os.Process
	procExited := make(chan error, 1)

	var restart <-chan time.Time
	var kill <-chan time.Time
	const killDelay = 5 * time.Second
	// long initial delay, will be reset to a proper interval when active
	healthCheck := time.NewTicker(time.Hour)

	for {
		select {
		case cmd := <-c.cmds:
			switch cmd {
			case childStart:
				switch status.State {
				case api.ChildStopped:
					curExec = 0
					if len(c.def.Init) != 0 {
						panic("unimplemented")
					}
					s := curStatus()
					var startErr error
					curProc, startErr = c.start(c.def.Name, c.def.Main, procExited)
					if startErr != nil {
						*s = api.ExecStatus{
							State:    api.ExecNotStarted,
							StartErr: startErr.Error(),
						}
						status.State = api.ChildError
					} else {
						*s = api.ExecStatus{
							State: api.ExecRunning,
							Pid:   curProc.Pid,
						}
						status.State = api.ChildRunning
					}
				default:
					panic("unimplemented")
				}
			case childStop:
				if curProc == nil {
					break
				}
				c.terminate(curProc, curStatus())
				kill = time.After(killDelay)
				status.State = api.ChildStopping
			case childDelete:
				panic("unimplemented")
			}
		case <-kill:
			if curProc == nil {
				break
			}
			c.kill(curProc, curStatus())
			// should already be in this state
			status.State = api.ChildStopping
		case err := <-procExited:
			if curProc == nil {
				break
			}
			s := curStatus()
			s.State = api.ExecEnded
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				s.ExitCode = ee.ExitCode()
			} else {
				s.ExitCode = 0
			}
			log.Printf("child %s pid %d exited with code %d", c.def.Name, s.Pid, s.ExitCode)
			s.Pid = 0
			switch status.State {
			case api.ChildStopping:
				// stop completed
				status.State = api.ChildStopped
			default:
				panic("unimplemented")
			}
		case <-restart:
			panic("unimplemented")
		case <-healthCheck.C:
			if c.def.HealthCheck == nil {
				continue
			}
			panic("unimplemented")
		}
		c.status.Store(cloneStatus(status))
	}
}

func initialStatus(c *child) api.ChildStatus {
	s := api.ChildStatus{
		State:  api.ChildStopped,
		Health: api.HealthStatus{},
		Init:   make([]api.ExecStatus, len(c.def.Init)),
		Main: api.ExecStatus{
			State: api.ExecNotStarted,
		},
	}
	for i := range s.Init {
		s.Init[i].State = api.ExecNotStarted
	}
	return s
}

func (c *child) start(
	name string,
	e api.Exec,
	exited chan<- error,
) (*os.Process, error) {
	cmd := exec.Command(e.Cmd, e.Args...)
	if e.Cwd != "" {
		cmd.Dir = e.Cwd
	}
	cmd.Env = os.Environ()
	for k, v := range e.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	// set pgid so we can kill process groups
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Noctty: true}
	// TODO: logfiles
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		err := cmd.Wait()
		exited <- err
	}()
	if err := isolateProcess(context.TODO(), name, cmd.Process); err != nil {
		log.Printf("ERROR: failed to isolate process: %v", err)
	}
	return cmd.Process, nil
}

func (c *child) terminate(p *os.Process, s *api.ExecStatus) {
	// signal the whole process group
	if err := syscall.Kill(-p.Pid, syscall.SIGTERM); err != nil {
		log.Printf("failed to terminate %d: %v", p.Pid, err)
	}
	s.State = api.ExecStopping
}

func (c *child) kill(p *os.Process, s *api.ExecStatus) {
	// signal the whole process group
	if err := syscall.Kill(-p.Pid, syscall.SIGKILL); err != nil {
		log.Printf("failed to kill %d: %v", p.Pid, err)
	}
	s.State = api.ExecStopping
}

func cloneStatus(s api.ChildStatus) *api.ChildStatus {
	r := s
	r.Init = slices.Clone(s.Init)
	return &r
}

func (c *child) Status() api.ChildStatus {
	return *cloneStatus(*c.status.Load())
}
