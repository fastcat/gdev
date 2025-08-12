package server

import (
	"os"
	"testing"

	"fastcat.org/go/gdev/internal"
)

func TestMain(m *testing.M) {
	// allow tests to access AppName and such
	internal.SetAppName("test")
	internal.LockCustomizations()
	os.Exit(m.Run()) //nolint:forbidigo // entrypoint
}
