package server

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"fastcat.org/go/gdev/addons/pm/api"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/lib/sys"
)

type child struct {
	def      api.Child
	status   atomic.Pointer[api.ChildStatus]
	cmds     chan childCmd
	wg       sync.WaitGroup
	isolator sys.Isolator

	restartDelay               time.Duration
	killDelay                  time.Duration
	healthCheckInitialInterval time.Duration
	healthCheckInterval        time.Duration
}

func newChild(def api.Child, isolator sys.Isolator) *child {
	c := &child{
		def:      def,
		cmds:     make(chan childCmd), // important that this be un-buffered
		isolator: isolator,

		// tests may override these
		restartDelay: time.Second, // TODO: scale
		killDelay:    5 * time.Second,
		// long initial delay, will be reset to a proper interval when active
		healthCheckInitialInterval: time.Second,
		healthCheckInterval:        10 * time.Second,
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

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

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
	healthCheck := time.NewTicker(time.Hour)
	healthCheck.Stop()
	defer healthCheck.Stop()
	healthChecks := -1
	healthResults := make(chan bool, 1)

	// usageTimer collects updated CPU usage for running processes
	usageTimer := time.NewTicker(time.Second)
	defer usageTimer.Stop()
	const usageLogInterval = 30 * time.Second
	usageLogTimer := time.NewTicker(usageLogInterval)
	defer usageLogTimer.Stop()
	lastUsageLogged := time.Now()
	lastTotalCPU := time.Duration(0)
	lastMem, lastMemPeak := uint64(0), uint64(0)
	usageLogDue := false

	reset := func() {
		curExec = 0
		for i := range status.Init {
			status.Init[i].Usage = nil
		}
		status.Main.Usage = nil
		lastUsageLogged = time.Now()
		lastTotalCPU = 0
		usageLogTimer.Reset(usageLogInterval)
	}

MANAGER:
	for {
		select {
		case cmd := <-c.cmds:
			switch cmd {
			case childStart:
				switch status.State {
				case api.ChildStopped, api.ChildError, api.ChildInitError, api.ChildDone:
					// start over from scratch
					reset()
					s := curStatus()
					curProc, *s, status.State = c.start(ctx, curExec, procExited)
				default:
					log.Printf("cannot start child %s from state %s", c.def.Name, status.State)
				}
			case childStop:
				if curProc == nil {
					switch status.State {
					case api.ChildError, api.ChildInitError:
						reset()
						// cancel any restart
						status.State = api.ChildStopped
					case api.ChildStopped, api.ChildDone:
						// ok
					default:
						// this is weird
						log.Printf("nothing to stop for child %s in state %s?", c.def.Name, status.State)
					}
					break
				}
				c.terminate(ctx, curProc, curStatus())
				kill = time.After(c.killDelay)
				status.State = api.ChildStopping
			case childDelete:
				if status.State != api.ChildStopped && status.State != api.ChildDone {
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
			c.kill(ctx, curProc, curStatus())
			// should already be in this state
			status.State = api.ChildStopping
		case err := <-procExited:
			if curProc == nil {
				panic("unimplemented: wtf")
			}
			curProc = nil
			s := curStatus()
			// make sure any children that tried to fork off get caught and killed via
			// the cgroup, unless they managed to escape into a new cgroup
			c.cleanup(ctx, s)
			s.State = api.ExecEnded
			if ee, ok := errors.AsType[*exec.ExitError](err); ok {
				s.ExitCode = ee.ExitCode()
				if sys, ok := ee.Sys().(syscall.WaitStatus); ok && sys.Signaled() {
					s.ExitSignal = int(sys.Signal())
				}
			} else {
				s.ExitCode, s.ExitSignal = 0, 0
			}
			log.Printf("child %s pid %d exited with %s", c.def.Name, s.Pid, s.DescribeExit())
			s.Pid = 0
			switch status.State {
			case api.ChildStopping:
				// re-check all the isolation groups to make sure all processes are
				// killed and cgroups removed
				c.cleanupAll(ctx, &status)
				// stop completed
				status.State = api.ChildStopped
				// reset the starting process to the beginning, but not the usage
				curExec = 0
			case api.ChildInitRunning:
				if s.ExitCode == 0 {
					log.Printf("child %s init %d complete, moving on", c.def.Name, curExec)
					// start next container
					curExec++
					s := curStatus()
					curProc, *s, status.State = c.start(ctx, curExec, procExited)
				} else {
					status.State = api.ChildInitError
					if c.def.NoRestart {
						log.Printf(
							"child %s init %d failed with %s, will not automatically restart",
							c.def.Name, curExec, s.DescribeExit(),
						)
					} else {
						log.Printf("child %s init %d failed with %s, will restart", c.def.Name, curExec, s.DescribeExit())
						restart = time.After(c.restartDelay)
					}
				}
			case api.ChildRunning:
				if c.def.OneShot {
					log.Printf("child %s one-shot completed with %s", c.def.Name, s.DescribeExit())
					if s.ExitCode == 0 {
						status.State = api.ChildDone
					} else {
						status.State = api.ChildError
					}
				} else {
					// treat this as an error except if no-restart exits successfully
					if s.ExitCode == 0 && c.def.NoRestart {
						status.State = api.ChildStopped
					} else {
						status.State = api.ChildError
					}
					if c.def.NoRestart {
						log.Printf(
							"child %s exited with %s, will not automatically restart",
							c.def.Name, s.DescribeExit(),
						)
					} else {
						log.Printf("child %s service exited with %s, will restart", c.def.Name, s.DescribeExit())
						restart = time.After(c.restartDelay)
					}
				}
			default:
				log.Printf("wtf? child %s got exit notification in state %s", c.def.Name, status.State)
			}
		case <-restart:
			log.Printf("child %s exec %d: restarting", c.def.Name, curExec)
			s := curStatus()
			curProc, *s, status.State = c.start(ctx, curExec, procExited)
		case <-healthCheck.C:
			// TODO: do a health check
			switch {
			case c.def.HealthCheck.Http != nil:
				timeout := time.Second
				if c.def.HealthCheck.TimeoutSeconds > 0 {
					timeout = time.Duration(c.def.HealthCheck.TimeoutSeconds) * time.Second
				}
				go func() { healthResults <- c.httpCheck(ctx, c.def.HealthCheck.Http, timeout) }()
			default:
				log.Printf("child %s: no recognized health check", c.def.Name)
			}

			healthChecks++
			// switch to the slower interval after N attempts
			if healthChecks == 5 {
				healthCheck.Reset(c.healthCheckInterval)
			}
		case healthy := <-healthResults:
			if status.Health.Healthy != healthy {
				desc := "healthy"
				if !healthy {
					desc = "unhealthy"
				}
				log.Printf("child %s is now %s", c.def.Name, desc)
			}
			status.Health.Healthy = healthy
			now := time.Now()
			if healthy {
				status.Health.LastHealthy = &now
			} else {
				status.Health.LastUnhealthy = &now
			}
		case <-usageTimer.C:
			// fall through to the bottom where we update the usage from all the
			// running processes
		case <-usageLogTimer.C:
			usageLogDue = true
		}

		// if the child main just started, activate the health-check timer
		if status.State == api.ChildRunning && c.def.HealthCheck != nil {
			if healthChecks < 0 {
				healthCheck.Reset(c.healthCheckInitialInterval)
				healthChecks = 0
			}
		} else {
			healthCheck.Stop()
			healthChecks = -1
		}

		for i := range status.Init {
			c.fillUsage(ctx, &status.Init[i])
		}
		c.fillUsage(ctx, &status.Main)

		if usageLogDue {
			now := time.Now()
			dur := now.Sub(lastUsageLogged)

			var total time.Duration
			memNow, memPeakNow := uint64(0), uint64(0)
			accumulate := func(s api.ExecStatus) {
				if u := s.Usage; u != nil {
					total += time.Duration(float64(time.Second) * (u.SystemSecs + u.UserSecs))
					if s.State == api.ExecRunning || s.State == api.ExecStopping {
						memNow += u.MemoryBytes
						memPeakNow += u.MemoryPeakBytes
					}
				}
			}
			for _, s := range status.Init {
				accumulate(s)
			}
			accumulate(status.Main)
			delta := total - lastTotalCPU
			// only log if it is using a "measurable" amount of CPU time
			if pct := 100.0 * delta.Seconds() / dur.Seconds(); pct >= 1 {
				log.Printf(
					"child %s total CPU usage over last %v: %v (%.2f%%)",
					c.def.Name, dur.Round(time.Millisecond), delta.Round(time.Millisecond),
					pct,
				)
			}

			// log memory usage if we got a new peak, or we have deviated by >= 10%
			// since the last memory log. This
			if memNow > 0 || memPeakNow > 0 {
				memDiffRat := math.Abs(float64(memNow-lastMem) / float64(max(lastMem, 1)))
				if memPeakNow != lastMemPeak || memDiffRat >= 0.1 {
					log.Printf(
						"child %s memory now=%d peak=%d",
						c.def.Name, memNow, memPeakNow,
					)
					lastMem, lastMemPeak = memNow, memPeakNow
				}

				lastTotalCPU = total
				lastUsageLogged = now
				usageLogDue = false
			}
		}

		c.status.Store(cloneStatus(status))
	}
}

func (c *child) fillUsage(ctx context.Context, s *api.ExecStatus) {
	if s.State != api.ExecRunning || s.Group == "" {
		return
	}
	if u, err := c.isolator.Usage(ctx, s.Group); err != nil {
		log.Printf("WARN: unable to get child usage for group %q: %v", s.Group, err)
	} else {
		s.Usage = &api.ExecUsage{
			UserSecs:        u.User.Seconds(),
			SystemSecs:      u.System.Seconds(),
			MemoryBytes:     u.Memory,
			MemoryPeakBytes: u.MemoryPeak,
		}
	}
}

func (c *child) httpCheck(ctx context.Context, check *api.HttpHealthCheck, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	u := &url.URL{
		// TODO: ipv6 hackery?
		Host: net.JoinHostPort("localhost", strconv.Itoa(check.Port)),
		Path: check.Path,
	}
	if check.Scheme != "" {
		u.Scheme = check.Scheme
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		log.Printf("failed to construct http req for %s: %v", c.def.Name, err)
		return false
	}
	client := http.DefaultClient
	if check.Insecure && check.Scheme == "https" {
		// TODO: make our own baseline transport if we can't clone the default one,
		// instead of panicing
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client = &http.Client{Transport: t}
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to send http req for %s: %v", c.def.Name, err)
		return false
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("http req for %s returned bad status %d", c.def.Name, resp.StatusCode)
		return false
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return true
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

var printCGroupsROWarning = sync.OnceFunc(func() {
	log.Print("process isolation failing due to read-only mount, suppressing errors")
})

func (c *child) start(
	ctx context.Context,
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
	// if logfile is not set, pass output to stdout/stderr and let journalctl
	// capture it. note that this only works if we're using systemd for isolation.
	if e.Logfile == "" {
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	} else {
		lf, err := os.OpenFile(e.Logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			log.Printf("failed to start %s, unable to open logfile %q: %v", c.def.Name, e.Logfile, err)
			return nil, api.ExecStatus{State: api.ExecNotStarted, StartErr: err.Error()}, errorState
		}
		cmd.Stdout, cmd.Stderr = lf, lf
		// the fd will be passed directly to the child, so we can close it when we return
		defer lf.Close() //nolint:errcheck
	}

	if err := cmd.Start(); err != nil {
		log.Printf("failed to start %s: %v", c.def.Name, err)
		return nil, api.ExecStatus{State: api.ExecNotStarted, StartErr: err.Error()}, errorState
	}
	log.Printf("started %s as pid %d", name, cmd.Process.Pid)
	c.wg.Go(func() {
		err := cmd.Wait()
		exited <- err
	})
	eStat := api.ExecStatus{
		State: api.ExecRunning,
		Pid:   cmd.Process.Pid,
	}
	if isolationGroup, err := c.isolator.Isolate(
		ctx,
		instance.AppName()+"-pm-"+name+".scope",
		cmd.Process,
	); err != nil {
		if errors.Is(err, syscall.EROFS) {
			printCGroupsROWarning()
		} else {
			log.Printf("ERROR: failed to isolate process %d as %q: %v", cmd.Process.Pid, name, err)
		}
	} else {
		eStat.Group = isolationGroup
		c.fillUsage(ctx, &eStat)
	}
	return cmd.Process, eStat, runningState
}

func (c *child) terminate(ctx context.Context, p *os.Process, s *api.ExecStatus) {
	c.fillUsage(ctx, s)
	// signal the whole process group
	if err := syscall.Kill(-p.Pid, syscall.SIGTERM); err != nil {
		log.Printf("failed to terminate %d: %v", p.Pid, err)
	} else {
		log.Printf("sent SIGTERM to child %s pid %d", c.def.Name, p.Pid)
	}

	s.State = api.ExecStopping
}

func (c *child) kill(ctx context.Context, p *os.Process, s *api.ExecStatus) {
	c.fillUsage(ctx, s)
	log.Printf("resorting to SIGKILL for child %s pid %d", c.def.Name, p.Pid)
	// signal the whole process group
	if err := syscall.Kill(-p.Pid, syscall.SIGKILL); err != nil {
		log.Printf("failed to kill %d: %v", p.Pid, err)
	}
	if s.Group != "" {
		if err := c.isolator.Cleanup(ctx, s.Group); err != nil {
			log.Printf("failed to cleanup isolation group %q: %v", s.Group, err)
		}
	}
	s.State = api.ExecStopping
}

func (c *child) cleanup(ctx context.Context, s *api.ExecStatus) {
	if s.Group == "" {
		return
	}
	if err := c.isolator.Cleanup(ctx, s.Group); err != nil {
		log.Printf("failed to cleanup isolation group %q: %v", s.Group, err)
	} else {
		s.Group = ""
	}
}

func (c *child) cleanupAll(ctx context.Context, s *api.ChildStatus) {
	for i := range s.Init {
		c.cleanup(ctx, &s.Init[i])
	}
	c.cleanup(ctx, &s.Main)
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
