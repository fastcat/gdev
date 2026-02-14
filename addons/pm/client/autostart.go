package client

import (
	"context"
	"os"
	"strings"

	"fastcat.org/go/gdev/addons/pm/api"
	"fastcat.org/go/gdev/lib/sys"
)

var DaemonFallbackLogFile string

func AutoStart(ctx context.Context, client api.API) error {
	if err := client.Ping(ctx); err == nil {
		// this is an uninteresting common case, no need to print a message here
		// fmt.Println("pm is already running")
		return nil
	}

	path := os.Args[0]
	args := []string{"pm", "daemon"}

	var env map[string]string
	if DaemonFallbackLogFile != "" {
		envList := os.Environ()
		env = make(map[string]string, len(envList)+1)
		for _, e := range envList {
			k, v, _ := strings.Cut(e, "=")
			env[k] = v
		}
		env[sys.FallbackLogFileEnv] = DaemonFallbackLogFile
	}

	// will become unit {appname}-pm.service
	return sys.StartDaemon(ctx, "pm", path, args, env)
}
