package docker

import (
	"fmt"

	"github.com/docker/docker/client"
)

func NewClient() (*client.Client, error) {
	c, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return c, nil
}
