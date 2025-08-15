package bootstrap

import (
	"context"

	"github.com/spf13/cobra"
)

const defaultCmdShort = "install & configure system dependencies"

func runnerCmd(plan *Plan) *cobra.Command {
	dryRun := false
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Short: defaultCmdShort,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dryRun {
				return SimPlan(cmd.Context(), plan)
			}
			return RunPlan(cmd.Context(), plan)
		},
	}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", dryRun, "don't actually change anything")
	return cmd
}

func RunPlan(ctx context.Context, plan *Plan) error {
	plan.AddDefaultSteps()
	return plan.Run(ctx)
}

func SimPlan(ctx context.Context, plan *Plan) error {
	plan.AddDefaultSteps()
	return plan.Sim(ctx)
}
