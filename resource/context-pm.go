package resource

import (
	"context"

	"fastcat.org/go/gdev/pm/api"
	"fastcat.org/go/gdev/pm/client"
)

func newPMClient(context.Context) (api.API, error) {
	c := client.NewHTTP()
	// this may be called before the pm daemon is started, so we don't ping here
	return c, nil
}

func init() {
	AddContextEntry(newPMClient)
}
