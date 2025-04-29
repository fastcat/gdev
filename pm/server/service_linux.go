package server

import (
	"context"
	"fmt"
	"os"

	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"

	"fastcat.org/go/gdev/instance"
)

func isolateProcess(
	ctx context.Context,
	name string,
	process *os.Process,
) error {
	conn, err := dbus.NewUserConnectionContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close() // nolint:errcheck
	ch := make(chan string, 1)
	_, err = conn.StartTransientUnitContext(
		ctx,
		instance.AppName()+"-pm-"+name+".scope",
		"fail", // error if unit already exists
		[]dbus.Property{
			dbus.PropDescription(fmt.Sprintf("%s pm service %s", instance.AppName(), name)),
			// auto-harvest the transient unit once all its processes exit
			{Name: "CollectMode", Value: godbus.MakeVariant("inactive-or-failed")},
			// put the given PID into it now
			dbus.PropPids(uint32(process.Pid)),
			// doesn't work: `Unit name 'xxx-pm.service' is not a slice`
			// dbus.PropSlice(instance.AppName() + "-pm.service"),
		},
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
		return fmt.Errorf("service isolation for %s (%d) failed: %s", name, process.Pid, status)
	}
}
