package resource

import (
	"context"
	"fmt"
	"time"

	"fastcat.org/go/gdev/addons/pm/api"
	"fastcat.org/go/gdev/addons/pm/client"
)

// Create a new PM client, auto-starting the PM daemon if necessary.
//
// Do not use this function directly, with the PM addon registered, this will be
// registered as a resource context value provider, fetch the existing client
// from the resource context.
func NewPMClient(ctx context.Context) (api.API, error) {
	c := client.NewHTTP()
	if err := client.AutoStart(ctx, c); err != nil {
		return nil, err
	}
	// it may take a moment to start up
	stop := time.After(5 * time.Second)
	retry := time.NewTicker(100 * time.Millisecond)
	var err error
	for {
		err = c.Ping(ctx)
		if err == nil {
			return c, nil
		}
		select {
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		case <-retry.C:
			// continue
		case <-stop:
			return nil, fmt.Errorf("pm daemon never became ready")
		}
	}
}
