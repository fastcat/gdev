package client

import (
	"context"
	"fmt"
	"os"

	"fastcat.org/go/gdev/addons/pm/api"
	"fastcat.org/go/gdev/lib/sys"
)

func AutoStart(ctx context.Context, client api.API) error {
	if err := client.Ping(ctx); err == nil {
		fmt.Println("pm is already running")
		return nil
	}

	path := os.Args[0]
	args := []string{"pm", "daemon"}

	// will become unit {appname}-pm.service
	return sys.StartDaemon(ctx, "pm", path, args, nil)
}
