package client

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fastcat.org/go/gdev/pm/api"
)

func TestHttp_Ping(t *testing.T) {
	// TODO: tests should not try to use the real socket path
	a := api.ListenAddr()
	if au, _ := a.(*net.UnixAddr); au != nil {
		_ = os.Remove(a.String())
		t.Cleanup(func() { _ = os.Remove(a.String()) })
	}
	l, err := net.Listen(a.Network(), a.String())
	require.NoError(t, err)
	hits := 0
	s := &httptest.Server{
		Listener: l,
		Config: &http.Server{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				w.WriteHeader(http.StatusNoContent)
			}),
		},
	}
	t.Cleanup(s.Close)
	s.Start()

	c := NewHTTP()
	err = c.Ping(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, hits)
}
