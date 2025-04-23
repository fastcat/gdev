package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
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
	r, err := h.do(ctx, http.MethodGet, api.PathPing, nil)
	if err != nil {
		return err
	}
	if r.Body != nil {
		defer r.Body.Close() //nolint:errcheck
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			return err
		}
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
	r, err := h.do(ctx, http.MethodGet, api.PathSummary, nil)
	if err != nil {
		return nil, err
	}
	if r.Body == nil {
		return nil, fmt.Errorf("%s: no response body", api.PathSummary)
	}
	defer r.Body.Close() //nolint:errcheck
	return readOne[[]api.ChildSummary](r.Body)
}

var ErrTrailingGarbage = errors.New("trailing garbage (JSON)")

func readOne[T any](r io.Reader) (T, error) {
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	var value T
	if err := d.Decode(&value); err != nil {
		return value, err
	}
	if d.More() {
		return value, ErrTrailingGarbage
	}
	if err := v.Struct(value); err != nil {
		return value, err
	}
	return value, nil
}

func (h *HTTP) do(
	ctx context.Context,
	method string,
	path string,
	reqBody io.Reader,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, h.url(path), reqBody)
	if err != nil {
		return nil, err
	}
	res, err := h.c().Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, heFromResp(res, path)
	}
	return res, nil

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
	u.Path = path.Clean(u.Path)
	return u.String()
}
