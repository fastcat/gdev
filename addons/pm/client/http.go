package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"fastcat.org/go/gdev/addons/pm/api"
	"fastcat.org/go/gdev/addons/pm/internal"
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
	_, err := h.do(ctx, http.MethodGet, api.PathPing, nil)
	return err
}

// Child implements api.API.
func (h *HTTP) Child(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	r, err := h.do(ctx, http.MethodGet, withPathValue(api.PathOneChild, api.PathChildParamName, name), nil)
	if err != nil {
		return nil, err
	}
	return internal.JSONBody[*api.ChildWithStatus](ctx, r.Body, "", true)
}

// DeleteChild implements api.API.
func (h *HTTP) DeleteChild(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	r, err := h.do(ctx, http.MethodDelete, withPathValue(api.PathOneChild, api.PathChildParamName, name), nil)
	if err != nil {
		return nil, err
	}
	return internal.JSONBody[*api.ChildWithStatus](ctx, r.Body, "", true)
}

// PutChild implements api.API.
func (h *HTTP) PutChild(ctx context.Context, child api.Child) (*api.ChildWithStatus, error) {
	body, err := json.Marshal(child)
	if err != nil {
		return nil, err
	}
	r, err := h.do(ctx, http.MethodPut, api.PathChild, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return internal.JSONBody[*api.ChildWithStatus](ctx, r.Body, "", true)
}

// StartChild implements api.API.
func (h *HTTP) StartChild(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	r, err := h.do(ctx, http.MethodPost, withPathValue(api.PathStartChild, api.PathChildParamName, name), nil)
	if err != nil {
		return nil, err
	}
	return internal.JSONBody[*api.ChildWithStatus](ctx, r.Body, "", true)
}

// StopChild implements api.API.
func (h *HTTP) StopChild(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	r, err := h.do(ctx, http.MethodPost, withPathValue(api.PathStopChild, api.PathChildParamName, name), nil)
	if err != nil {
		return nil, err
	}
	return internal.JSONBody[*api.ChildWithStatus](ctx, r.Body, "", true)
}

// Summary implements api.API.
func (h *HTTP) Summary(ctx context.Context) ([]api.ChildSummary, error) {
	r, err := h.do(ctx, http.MethodGet, api.PathSummary, nil)
	if err != nil {
		return nil, err
	}
	return internal.JSONBody[[]api.ChildSummary](ctx, r.Body, "dive", true)
}

// Terminate implements api.API.
func (h *HTTP) Terminate(ctx context.Context) error {
	_, err := h.do(ctx, http.MethodPost, api.PathTerminate, nil)
	return err
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

func withPathValue(
	match string,
	name string, //nolint:unparam // important for future use
	value string,
) string {
	return strings.Replace(match, "{"+name+"}", value, 1)
}
