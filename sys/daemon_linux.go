package sys

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"fastcat.org/go/gdev/instance"
	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"
)

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
	// run as a transient systemd service
	conn, err := dbus.NewUserConnectionContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close() // nolint:errcheck
	ch := make(chan string, 1)
	props := []dbus.Property{
		dbus.PropDescription(fmt.Sprintf("%s - %s", instance.AppName(), name)),
		{Name: "CollectMode", Value: godbus.MakeVariant("inactive-or-failed")},
		dbus.PropType("exec"),
		dbus.PropExecStart(append([]string{path}, args...), true),
	}
	if len(env) != 0 {
		envs := make([]string, 0, len(env))
		for k, v := range env {
			envs = append(envs, k+"="+v)
		}
		props = append(props, dbus.Property{
			Name:  "Environment",
			Value: godbus.MakeVariant(envs),
		})
	}
	_, err = conn.StartTransientUnitContext(
		ctx,
		fmt.Sprintf("%s-%s.service", instance.AppName(), name),
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
