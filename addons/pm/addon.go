package pm

import (
	"context"
	"time"

	"fastcat.org/go/gdev/addons"
	pmResource "fastcat.org/go/gdev/addons/pm/resource"
	"fastcat.org/go/gdev/addons/pm/server"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/resource"
)

var addon = addons.Addon[config]{
	Config: config{
		// placeholder
	},
}

type config struct {
	tasks []server.Task
}
type option func(*config)

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	addon.RegisterIfNeeded(addons.Definition{
		Name: "pm",
		Description: func() string {
			return "Process manager daemon"
		},
		Initialize: initialize,
	})
}

func initialize() error {
	instance.AddCommandBuilders(pmCmd)
	resource.AddContextEntry(pmResource.NewPMClient)
	return nil
}

// WithTask adds a periodic background task to the pm daemon.
//
// The task will run at the specified interval. If a run overruns the interval,
// the missed tick(s) will be skipped so that it doesn't run continuously.
//
// If the timeout is greater than zero, the context passed to the task will be
// canceled after the timeout, otherwise it will inherit the pm daemon's context
// that is canceled when the daemon is asked to shut down.
func WithTask(interval, timeout time.Duration, run func(ctx context.Context)) option {
	return func(c *config) {
		c.tasks = append(c.tasks, server.Task{
			Interval: interval,
			Timeout:  timeout,
			Run:      run,
		})
	}
}
