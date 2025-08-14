package github

import (
	"context"
	"encoding/json"
	"net/http"
	"path"
	"strings"

	"fastcat.org/go/gdev/lib/httpx"
)

// Client provides a rudimentary client for acccessing GitHub resources
// over its REST api, without pulling in large dependencies.
type Client struct {
	c *http.Client
}

func NewClient(opts ...ClientOpt) *Client {
	c := &Client{c: http.DefaultClient}
	for _, o := range opts {
		o(c)
	}
	return c
}

type ClientOpt func(*Client)

func WithToken(token string) ClientOpt {
	return func(c *Client) {
		if c.c == http.DefaultClient {
			c.c = &http.Client{Transport: http.DefaultTransport}
		}
		c.c.Transport = httpx.WithBearer(c.c.Transport, token)
	}
}

func (c *Client) Get(ctx context.Context, path string, respData any) error {
	if req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://api.github.com/"+strings.TrimPrefix(path, "/"),
		nil,
	); err != nil {
		return err
	} else if err := c.DoAndParse(req, respData); err != nil {
		return err
	}
	return nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.c.Do(req)
}

func (c *Client) DoAndParse(req *http.Request, respData any) error {
	resp, err := c.Do(req)
	if err != nil {
		return err
	} else if !httpx.IsHTTPOk(resp) {
		return httpx.HTTPResponseErr(resp, req.URL.Path) // TODO: contextual base message
	}
	defer resp.Body.Close() // nolint:errcheck
	d := json.NewDecoder(resp.Body)
	return d.Decode(respData)
}

func (c *Client) GetRelease(ctx context.Context, owner, repo, tag string) (*Release, error) {
	var urlPath string
	if tag == "latest" {
		// https://docs.github.com/en/rest/releases/releases?apiVersion=2022-11-28#get-the-latest-release
		urlPath = path.Join("/repos", owner, repo, "releases", "latest")
	} else {
		// https://docs.github.com/en/rest/releases/releases?apiVersion=2022-11-28#get-a-release-by-tag-name
		urlPath = path.Join("/repos", owner, repo, "releases", "tags", tag)
	}
	var resp Release
	if err := c.Get(ctx, urlPath, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Download(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// must set this else we get json description of the resource back instead of the binary content
	req.Header.Set("accept", "application/octet-stream")
	return c.Do(req)
}

type Release struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
	// very incomplete
}
type ReleaseAsset struct {
	URL string `json:"url"`
	// BrowserDownloadURL string `json:"browser_download_url"`

	Name        string `json:"name"`
	Label       string `json:"label"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`

	// very incomplete
}
