package resource

import (
	"context"
	"fmt"
	"time"

	"fastcat.org/go/gdev/pm"
	"fastcat.org/go/gdev/pm/api"
	"fastcat.org/go/gdev/pm/client"
)

func newPMClient(ctx context.Context) (api.API, error) {
	c := client.NewHTTP()
	if err := pm.AutoStart(ctx, c); err != nil {
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

func init() {
	AddContextEntry(newPMClient)
}
