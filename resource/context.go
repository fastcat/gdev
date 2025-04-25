package resource

import (
	"context"
	"errors"

	"fastcat.org/go/gdev/pm/api"
)

type pmk struct{}

func PMClient(ctx context.Context) api.API {
	val := ctx.Value(pmk{})
	if val == nil {
		return nil
	}
	return val.(api.API)
}

func WithPMClient(ctx context.Context, client api.API) context.Context {
	return context.WithValue(ctx, pmk{}, client)
}

var ErrNoPMClient = errors.New("no pm client available")
