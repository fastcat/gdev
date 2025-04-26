package pm

import (
	"context"
	"fmt"
	"os"

	"fastcat.org/go/gdev/pm/api"
	"fastcat.org/go/gdev/sys"
)

func AutoStart(ctx context.Context, client api.API) error {
	if err := client.Ping(ctx); err == nil {
		fmt.Println("pm is already running")
		return nil
	}

	path := os.Args[0]
	args := []string{"pm", "daemon"}

	return sys.StartDaemon(ctx, "pm", path, args, nil)
}
