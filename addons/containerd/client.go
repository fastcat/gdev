package containerd

import (
	"errors"
	"fmt"

	"fastcat.org/go/gdev/internal"
	"github.com/containerd/containerd/v2/client"
)

func NewClient() (*client.Client, error) {
	internal.CheckLockedDown()
	if config == nil {
		panic(errors.New("addon not configured"))
	}
	c, err := client.New(config.clientAddr, config.clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create containerd client: %w", err)
	}
	return c, nil
}
