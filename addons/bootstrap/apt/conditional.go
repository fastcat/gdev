package apt

import (
	"fmt"
	"strings"

	"fastcat.org/go/gdev/addons/bootstrap"
)

// AddPackageIfAvailable is like [bootstrap.AddAptPackagesStep], but will only add the
// package to the install list if it is available. To ensure accurate results,
// this always runs after the primary [bootstrap.StepNameAptUpdate] step. You can add
// additional after constraints if it needs to go after a secondary update step.
func AddPackageIfAvailable(stepName, packageName string) *bootstrap.Step {
	mark := func(ctx *bootstrap.Context) error {
		if avail, err := AptAvailable(ctx); err != nil {
			return err
		} else if _, ok := avail[packageName]; ok {
			AddPackages(ctx, packageName)
		} else {
			fmt.Printf("Package %s is not available, skipping\n", packageName)
		}
		return nil
	}
	return bootstrap.NewStep(
		stepName,
		mark,
		// won't be entirely accurate if run in sim due to maybe not having all the
		// apt data, but better than nothing
		bootstrap.SimFunc(mark),
		bootstrap.BeforeSteps(StepNameInstall),
		bootstrap.AfterSteps(StepNameUpdate),
	)
}

// AddFirstAvailable is like [AddPackageIfAvailable], but will add the first
// available package from the list of candidates. If none of the candidates are
// available, it will fail.
func AddFirstAvailable(
	stepName string,
	candidates ...string,
) *bootstrap.Step {
	mark := func(ctx *bootstrap.Context) error {
		avail, err := AptAvailable(ctx)
		if err != nil {
			return err
		}
		for _, pkg := range candidates {
			if _, ok := avail[pkg]; ok {
				// this will print a message, we don't need to
				AddPackages(ctx, pkg)
				return nil
			}
		}
		return fmt.Errorf(
			"no packages available from candidates %s",
			strings.Join(candidates, " "),
		)
	}
	return bootstrap.NewStep(
		stepName,
		mark,
		// same sim accuracy caveat as [AddPackageIfAvailable]
		bootstrap.SimFunc(mark),
		bootstrap.BeforeSteps(StepNameInstall),
		bootstrap.AfterSteps(StepNameUpdate),
	)
}
