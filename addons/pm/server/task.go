package server

import (
	"context"
	"sync"
	"time"
)

type Task struct {
	Interval time.Duration
	Timeout  time.Duration
	Run      func(ctx context.Context)
}

func (d *daemon) startTasks(ctx context.Context, wg *sync.WaitGroup) {
	for _, t := range d.tasks {
		wg.Go(func() {
			d.runTask(ctx, t)
		})
	}
}

func (d *daemon) runTask(ctx context.Context, t Task) {
	ticker := time.NewTicker(t.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.invokeTask(ctx, t)
			// if task over-runs the ticker, don't restart it immediately
			select {
			case <-ticker.C:
			default:
			}
		}
	}
}

func (d *daemon) invokeTask(ctx context.Context, t Task) {
	var cancel context.CancelFunc
	if t.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, t.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()
	t.Run(ctx)
}
