package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"path"
	"strings"

	"fastcat.org/go/gdev/lib/httpx"
)

// GitHubClient provides a rudimentary client for acccessing GitHub resources
// over its REST api, without pulling in large dependencies.
type GitHubClient struct {
	c *http.Client
}

func NewGitHubClient(opts ...ghClientOpt) *GitHubClient {
	c := &GitHubClient{c: http.DefaultClient}
	for _, o := range opts {
		o(c)
	}
	return c
}

type ghClientOpt func(*GitHubClient)

func WithToken(token string) ghClientOpt {
	return func(c *GitHubClient) {
		if c.c == http.DefaultClient {
			c.c = &http.Client{Transport: http.DefaultTransport}
		}
		c.c.Transport = withBearer(c.c.Transport, token)
	}
}

func (c *GitHubClient) Get(ctx context.Context, path string, respData any) error {
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

func (c *GitHubClient) Do(req *http.Request) (*http.Response, error) {
	return c.c.Do(req)
}

func (c *GitHubClient) DoAndParse(req *http.Request, respData any) error {
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

func (c *GitHubClient) Release(ctx context.Context, owner, repo, tag string) (*GitHubRelease, error) {
	var urlPath string
	if tag == "latest" {
		// https://docs.github.com/en/rest/releases/releases?apiVersion=2022-11-28#get-the-latest-release
		urlPath = path.Join("/repos", owner, repo, "releases", "latest")
	} else {
		// https://docs.github.com/en/rest/releases/releases?apiVersion=2022-11-28#get-a-release-by-tag-name
		urlPath = path.Join("/repos", owner, repo, "releases", "tags", tag)
	}
	var resp GitHubRelease
	if err := c.Get(ctx, urlPath, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *GitHubClient) Download(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// must set this else we get json description of the resource back instead of the binary content
	req.Header.Set("accept", "application/octet-stream")
	return c.Do(req)
}

type GitHubRelease struct {
	TagName string               `json:"tag_name"`
	Assets  []GitHubReleaseAsset `json:"assets"`
	// very incomplete
}
type GitHubReleaseAsset struct {
	URL string `json:"url"`
	// BrowserDownloadURL string `json:"browser_download_url"`

	Name        string `json:"name"`
	Label       string `json:"label"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`

	// very incomplete
}
