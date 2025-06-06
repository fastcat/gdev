package server

import (
	"os"
	"testing"

	"fastcat.org/go/gdev/internal"
)

func TestMain(m *testing.M) {
	// allow tests to access AppName and such
	internal.LockCustomizations()
	os.Exit(m.Run())
}
