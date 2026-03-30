package docker

import (
	"fmt"

	"github.com/moby/moby/client"
)

func NewClient() (*client.Client, error) {
	c, err := client.New(
		client.FromEnv,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return c, nil
}
