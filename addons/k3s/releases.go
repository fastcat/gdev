package k3s

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"fastcat.org/go/gdev/internal"
)

// partial structure for parsing https://update.k3s.io/v1-release/channels
type k3sChannels struct {
	Data []k3sChannel `json:"data"`
}

// partial structure
type k3sChannel struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Latest string `json:"latest"`
}

func getK3SChannels(
	ctx context.Context,
	c *http.Client,
) (*k3sChannels, error) {
	if c == nil {
		c = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		`https://update.k3s.io/v1-release/channels`,
		nil,
	)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	if !internal.IsHTTPOk(resp) {
		return nil, internal.HTTPResponseErr(resp, "failed to query k3s release channels")
	}

	d := json.NewDecoder(resp.Body)
	var channels k3sChannels
	if err := d.Decode(&channels); err != nil {
		return nil, fmt.Errorf("failed parsing k3s channel data: %w", err)
	}
	return &channels, nil
}

func (c *k3sChannels) channel(id string) *k3sChannel {
	for i := range c.Data {
		if c.Data[i].ID == id {
			return &c.Data[i]
		}
	}
	return nil
}
