package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"fastcat.org/go/gdev/pm/api"
)

func NewHTTP() *HTTP {
	t := &http.Transport{
		// select defaults copied from Go 1.24.2
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	t.DialContext = defaultDialer
	c := &http.Client{Transport: t}
	return &HTTP{Client: c}
}

type HTTP struct {
	Client *http.Client
	Base   *url.URL
}

var _ api.API = (*HTTP)(nil)

// Ping implements api.API.
func (h *HTTP) Ping(ctx context.Context) error {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url(api.PathPing), nil)
	if err != nil {
		return err
	}
	res, err := h.c().Do(r)
	if err != nil {
		return err
	}
	if res.Body != nil {
		defer res.Body.Close() //nolint:errcheck
		if _, err := io.Copy(io.Discard, res.Body); err != nil {
			return err
		}
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("ping returned http status %d", res.StatusCode)
	}
	return nil
}

// Child implements api.API.
func (h *HTTP) Child(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	panic("unimplemented")
}

// DeleteChild implements api.API.
func (h *HTTP) DeleteChild(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	panic("unimplemented")
}

// PutChild implements api.API.
func (h *HTTP) PutChild(ctx context.Context, child api.Child) (*api.ChildWithStatus, error) {
	panic("unimplemented")
}

// StartChild implements api.API.
func (h *HTTP) StartChild(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	panic("unimplemented")
}

// StopChild implements api.API.
func (h *HTTP) StopChild(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	panic("unimplemented")
}

// Summary implements api.API.
func (h *HTTP) Summary(ctx context.Context) ([]api.ChildSummary, error) {
	panic("unimplemented")
}

func (h *HTTP) c() *http.Client {
	if h.Client != nil {
		return h.Client
	}
	return http.DefaultClient
}

func (h *HTTP) url(p string) string {
	var u *url.URL
	if h.Base != nil {
		u = h.Base
	} else {
		u = &url.URL{
			Scheme: "http",
			Host:   "localhost",
			Path:   "/",
		}
	}
	u = u.ResolveReference(&url.URL{Path: "/./" + p})
	return u.String()
}
