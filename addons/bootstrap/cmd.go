package bootstrap

import "context"

func Run(ctx context.Context) error {
	return defaultPlan.Run(ctx)
}
