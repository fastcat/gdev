package sys

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"

	"fastcat.org/go/gdev/instance"
)

// FallbackLogFileEnv is an environment variable that can be set to provide
// a fallback log file path for daemons started without systemd support.
//
// This will not be passed to the actual daemon.
//
// TODO: this is ugly
const FallbackLogFileEnv = "__FALLBACK_LOG_FILE"

var expectSystemdAbsent = false

// ExpectSystemdAbsent can be called from apps that know they are running in an
// environment where the systemd user instance is not expected to be available.
// It will not prevent attempting to use it, but it will suppress warnings if it
// is absent.
func ExpectSystemdAbsent() {
	expectSystemdAbsent = true
}

func StartDaemon(
	ctx context.Context,
	name string,
	path string,
	args []string,
	env map[string]string,
) error {
	// systemd requires an abs path for the exec
	if !filepath.IsAbs(path) {
		var pathErr error
		if path, pathErr = exec.LookPath(path); pathErr != nil {
			return fmt.Errorf("cannot resolve daemon path %q: %w", path, pathErr)
		}
		// LookPath won't deal with things like "./foo", so we need a second pass to
		// fix those up
		if !filepath.IsAbs(path) {
			if path, pathErr = filepath.Abs(path); pathErr != nil {
				return fmt.Errorf("cannot daemon path %q absolute: %w", path, pathErr)
			}
		}
	}

	var envs []string
	var fallbackLogFile string
	if len(env) != 0 {
		envs = make([]string, 0, len(env))
		for k, v := range env {
			if k == FallbackLogFileEnv {
				fallbackLogFile = v
				continue
			}
			envs = append(envs, k+"="+v)
		}
	}
	unitName := instance.AppName() + "-" + name + ".service"
	// run as a transient systemd service
	conn, err := SystemdUserConn(ctx)
	if err != nil {
		if !expectSystemdAbsent {
			fmt.Fprintf(os.Stderr,
				"WARNING: can't start %s daemon via systemd, falling back on manual cgroups isolation: %v\n",
				name, err,
			)
		}
		return startDaemonNoSystemd(ctx, unitName, path, args, envs, fallbackLogFile)
	}
	defer conn.Close() // nolint:errcheck

	ch := make(chan string, 1)
	props := []dbus.Property{
		dbus.PropDescription(fmt.Sprintf("%s - %s", instance.AppName(), name)),
		{Name: "CollectMode", Value: godbus.MakeVariant("inactive-or-failed")},
		dbus.PropType("exec"),
		dbus.PropExecStart(append([]string{path}, args...), true),
	}
	if len(envs) != 0 {
		props = append(props, dbus.Property{
			Name:  "Environment",
			Value: godbus.MakeVariant(envs),
		})
	}
	_, err = conn.StartTransientUnitContext(
		ctx,
		unitName,
		"fail", // error if already exists
		props,
		ch,
	)
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
		return fmt.Errorf("daemon start for %s failed: %s", name, status)
	}
}

func startDaemonNoSystemd(
	ctx context.Context,
	unitName string,
	path string,
	args []string,
	envs []string,
	fallbackLogFile string,
) error {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", os.DevNull, err)
	}
	defer devNull.Close() //nolint:errcheck

	out := devNull
	if fallbackLogFile != "" {
		lf, err := os.OpenFile(fallbackLogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open fallback log file %q: %w", fallbackLogFile, err)
		}
		defer lf.Close() //nolint:errcheck
		out = lf
	}

	proc, err := os.StartProcess(
		path,
		append([]string{path}, args...),
		&os.ProcAttr{
			Env: envs,
			Files: []*os.File{
				devNull, // stdin
				out,     // stdout
				out,     // stderr
			},
			Sys: &syscall.SysProcAttr{
				Setsid: true,
				// can't setpgid if we are already setsid
				// noctty is unnecessary since we have /dev/null as stdio
				// PidFD: &pidFD,
			},
		},
	)
	if err != nil {
		return err
	}
	// TODO: re-use getIsolator() instance here, since it _should_ be a cgroups one
	if _, err := (&cgroupsIsolator{}).Isolate(ctx, unitName, proc); err != nil {
		return err
	}

	return nil
}
