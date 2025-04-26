package server

import (
	"context"
	"fmt"
	"os"

	"fastcat.org/go/gdev/instance"
	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"
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
