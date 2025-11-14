package sys

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemdUserConn(t *testing.T) {
	// TODO: this is a poor way to detect that we are inside a container
	cgroup, err := os.ReadFile("/proc/self/cgroup")
	require.NoError(t, err)
	inContainer := strings.TrimSpace(string(cgroup)) == "0::/"

	conn, err := SystemdUserConn(t.Context())
	t.Cleanup(func() {
		if conn != nil {
			conn.Close()
		}
	})

	t.Run("on host", func(t *testing.T) {
		if inContainer {
			t.SkipNow()
		}
		// TODO: this assumes systemd is available
		assert.NoError(t, err)
	})

	t.Run("in container", func(t *testing.T) {
		if !inContainer {
			t.SkipNow()
		}
		// TODO: this assumes systemd isn't running inside the container
		assert.ErrorIs(t, err, ErrWrongNamespace)
	})
}
