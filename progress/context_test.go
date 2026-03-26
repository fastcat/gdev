package progress

import "testing"

func TestStartWriter(t *testing.T) {
	ctx, stop := StartWriter(t.Context())
	t.Cleanup(stop)
	t.Log("OK")
	_ = ctx
	stop()
	t.Log("stopped")
}
