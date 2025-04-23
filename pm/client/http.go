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

func NewHttp() *Http {
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
	return &Http{Client: c}
}

type Http struct {
	Client *http.Client
	Base   *url.URL
}

var _ api.Client = (*Http)(nil)

// Ping implements api.Client.
func (h *Http) Ping(ctx context.Context) error {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url(api.PathPing), nil)
	if err != nil {
		return err
	}
	res, err := h.Client.Do(r)
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

// Child implements api.Client.
func (h *Http) Child(ctx context.Context, name string) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// DeleteChild implements api.Client.
func (h *Http) DeleteChild(ctx context.Context, name string) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// PutChild implements api.Client.
func (h *Http) PutChild(ctx context.Context, child api.Child) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// StartChild implements api.Client.
func (h *Http) StartChild(ctx context.Context, name string) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// StopChild implements api.Client.
func (h *Http) StopChild(ctx context.Context, name string) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// Summary implements api.Client.
func (h *Http) Summary(ctx context.Context) ([]api.ChildSummary, error) {
	panic("unimplemented")
}

func (h *Http) c() *http.Client {
	if h.Client != nil {
		return h.Client
	}
	return http.DefaultClient
}

func (h *Http) url(p string) string {
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
