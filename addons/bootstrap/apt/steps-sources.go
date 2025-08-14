package apt

import (
	"fastcat.org/go/gdev/addons/bootstrap"
)

// SourceInstallStep creates a bootstrap step that installs the given APT
// source.
//
// You should always adjust the returned step with [bootstrap.BeforeSteps],
// commonly [bootstrap.StepNameAptUpdate].
func SourceInstallStep(
	installer *SourceInstaller,
	opts ...bootstrap.StepOpt,
) *bootstrap.Step {
	return bootstrap.NewStep(
		"Install APT source "+installer.SourceName,
		func(ctx *bootstrap.Context) error {
			if _, err := installer.Install(ctx); err != nil {
				return err
			}
			ChangedSources(ctx)
			return nil
		},
	).With(
		bootstrap.SimFunc(func(ctx *bootstrap.Context) error {
			if _, err := installer.Sim(ctx); err != nil {
				return err
			}
			ChangedSources(ctx)
			return nil
		}),
	).With(opts...)
}

// PublicSourceInstallSteps creates a slice of bootstrap steps that install the
// given APT sources before the initial APT update step, when only (and all)
// public sources are presumed available.
func PublicSourceInstallSteps(
	installers ...*SourceInstaller,
) []*bootstrap.Step {
	steps := make([]*bootstrap.Step, 0, len(installers))
	for _, installer := range installers {
		steps = append(steps, SourceInstallStep(
			installer,
			bootstrap.BeforeSteps(StepNameUpdate),
		))
	}
	return steps
}
