package bootstrap

import "context"

func RunPlan(ctx context.Context, plan *Plan) error {
	plan.AddDefaultSteps()
	return plan.Run(ctx)
}

func SimPlan(ctx context.Context, plan *Plan) error {
	plan.AddDefaultSteps()
	return plan.Sim(ctx)
}
