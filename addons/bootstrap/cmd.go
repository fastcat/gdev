package bootstrap

import "context"

func Run(ctx context.Context) error {
	defaultPlan.AddDefaultSteps()
	return defaultPlan.Run(ctx)
}
