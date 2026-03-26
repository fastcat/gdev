package progress

import (
	"context"
	"runtime"
	"sync"

	"github.com/jedib0t/go-pretty/v6/progress"
)

type writerKey struct{}

func WithWriter(ctx context.Context, w progress.Writer) context.Context {
	return context.WithValue(ctx, writerKey{}, w)
}

func ContextWriter(ctx context.Context) progress.Writer {
	w, _ := ctx.Value(writerKey{}).(progress.Writer)
	return w
}

func StartWriter(ctx context.Context) (_ context.Context, stop func()) {
	if w := ContextWriter(ctx); w != nil {
		// don't create a duplicate one
		return ctx, func() {}
	}

	// progress has an internal context, but doesn't support setting it to base
	// off something other than context.Background()
	pw := progress.NewWriter()
	pw.SetStyle(progress.StyleBlocks)
	pw.SetTrackerPosition(progress.PositionRight)
	var wg sync.WaitGroup
	wg.Go(func() { pw.Render() })
	// if the caller's work all finishes before render starts, then Stop() might
	// not stop it, so we wait for it to confirm it's running before we return, so
	// that callers don't have to worry about this.
	for !pw.IsRenderInProgress() {
		// this REALLY should not take long, don't even bother sleeping, just yield
		// the scheduler so the other goroutine is sure to run ASAP.
		runtime.Gosched()
	}
	stop = func() {
		pw.Stop()
		wg.Wait()
	}
	return WithWriter(ctx, pw), stop
}

func AddTracker(ctx context.Context, t *progress.Tracker) {
	if w := ContextWriter(ctx); w != nil {
		w.AppendTracker(t)
	}
}
