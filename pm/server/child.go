package server

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"fastcat.org/go/gdev/pm/api"
)

type child struct {
	def    api.Child
	status atomic.Pointer[api.ChildStatus]
	cmds   chan childCmd
	wg     sync.WaitGroup
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
	// TODO: this is non-standard use of the waitgroup
	c.wg.Add(1)
	defer c.wg.Done()

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

	var kill <-chan time.Time
	var restart <-chan time.Time
	const restartDelay = time.Second // TODO: scale
	const killDelay = 5 * time.Second
	// long initial delay, will be reset to a proper interval when active
	healthCheck := time.NewTicker(time.Hour)
	defer healthCheck.Stop()

MANAGER:
	for {
		select {
		case cmd := <-c.cmds:
			switch cmd {
			case childStart:
				switch status.State {
				case api.ChildStopped, api.ChildError, api.ChildInitError:
					// start over from scratch
					curExec = 0
					s := curStatus()
					curProc, *s, status.State = c.start(curExec, procExited)
				default:
					log.Printf("cannot start child %s from state %s", c.def.Name, status.State)
				}
			case childStop:
				if curProc == nil {
					switch status.State {
					case api.ChildError, api.ChildInitError:
						curExec = 0
						// cancel any restart
						status.State = api.ChildStopped
					case api.ChildStopped:
						// ok
					default:
						// this is weird
						log.Printf("nothing to stop for child %s in state %s?", c.def.Name, status.State)
					}
					break
				}
				c.terminate(curProc, curStatus())
				kill = time.After(killDelay)
				status.State = api.ChildStopping
			case childDelete:
				if status.State != api.ChildStopped {
					log.Printf("cannot delete child %s in state %s", c.def.Name, status.State)
					break
				}
				// TODO: assert curProc != nil?
				break MANAGER
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
				panic("unimplemented: wtf")
			}
			curProc = nil
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
				// reset the starting process to the beginning
				curExec = 0
			case api.ChildInitRunning:
				if s.ExitCode == 0 {
					log.Printf("child %s init %d complete, moving on", c.def.Name, curExec)
					// start next container
					curExec++
					s := curStatus()
					curProc, *s, status.State = c.start(curExec, procExited)
				} else {
					log.Printf("child %s init %d failed with code %d, will restart", c.def.Name, curExec, s.ExitCode)
					status.State = api.ChildInitError
					restart = time.After(restartDelay)
				}
			case api.ChildRunning:
				// TODO: one-shot support
				log.Printf("child %s service exited with code %d, will restart", c.def.Name, s.ExitCode)
				// treat this as an error
				status.State = api.ChildError
				restart = time.After(restartDelay)
			default:
				log.Printf("wtf? child %s got exit notification in state %s", c.def.Name, status.State)
			}
		case <-restart:
			log.Printf("child %s exec %d: restarting", c.def.Name, curExec)
			s := curStatus()
			curProc, *s, status.State = c.start(curExec, procExited)
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
	idx int,
	exited chan<- error,
) (*os.Process, api.ExecStatus, api.ChildState) {
	runningState, errorState := api.ChildRunning, api.ChildError
	e := c.def.Main
	name := c.def.Name
	if idx < len(c.def.Init) {
		runningState, errorState = api.ChildInitRunning, api.ChildInitError
		e = c.def.Init[idx]
		name = c.def.Name + "-init-" + strconv.Itoa(idx)
	}
	cmd := exec.Command(e.Cmd, e.Args...)
	if e.Cwd != "" {
		cmd.Dir = e.Cwd
	}
	cmd.Env = os.Environ()
	for k, v := range e.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	// set pgid so we can kill process groups
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// TODO: logfiles
	if err := cmd.Start(); err != nil {
		return nil, api.ExecStatus{State: api.ExecNotStarted, StartErr: err.Error()}, errorState
	}
	log.Printf("started %s as pid %d", name, cmd.Process.Pid)
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		err := cmd.Wait()
		exited <- err
	}()
	if err := isolateProcess(context.TODO(), name, cmd.Process); err != nil {
		log.Printf("ERROR: failed to isolate process %d as %q: %v", cmd.Process.Pid, name, err)
	}
	return cmd.Process,
		api.ExecStatus{
			State: api.ExecRunning,
			Pid:   cmd.Process.Pid,
		},
		runningState
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

func (c *child) Wait() {
	c.wg.Wait()
}
