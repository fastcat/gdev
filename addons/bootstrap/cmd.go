package bootstrap

import "context"

func Run(ctx context.Context) error {
	addon.Config.plan.AddDefaultSteps()
	return addon.Config.plan.Run(ctx)
}
