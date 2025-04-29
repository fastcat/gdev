package containerd

import (
	"fmt"

	"github.com/containerd/containerd/v2/client"
)

func NewClient() (*client.Client, error) {
	addon.CheckInitialized()
	c, err := client.New(addon.Config.clientAddr, addon.Config.clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create containerd client: %w", err)
	}
	return c, nil
}
