package apt

import (
	"fastcat.org/go/gdev/addons/bootstrap"
)

// SourceInstallStep creates a bootstrap step that installs the given APT
// source.
//
// You should always adjust the step with before/after constraints. For example
// if this is a public source, you would typically use [bootstrap.BeforeSteps],
// commonly [bootstrap.StepNameAptUpdate] (or call [PublicSourceInstallSteps]).
// If this is a private source, then you would typically use
// [bootstrap.AfterSteps] and [bootstrap.BeforeSteps] to ensure it runs after
// the normal apt setup and before the secondary apt install that uses packages
// from this private source.
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
