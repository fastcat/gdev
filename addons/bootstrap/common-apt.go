package bootstrap

import "fmt"

// Name of the step registered by [AddAptUpdate]. Steps that modify apt sources
// should reference this as a `before` constraint.
const StepNameAptUpdate = "apt-update"

func AddAptUpdate() {
	AddStep(aptUpdate())
}

func aptUpdate() *step {
	return Step(StepNameAptUpdate, doAptUpdate)
}

func doAptUpdate(ctx *Context) error {
	return Shell(
		ctx,
		[]string{"apt", "update"},
		WithSudo("update available packages"),
	)
}

// Name of the step registered by [AddAptInstall]. This step will install
// pending packages enqueued with [AddAptPackages]. Set any step that uses that
// to be before this step.
const StepNameAptInstall = "apt-install"

func AddAptInstall() {
	AddStep(aptInstall())
}
func aptInstall() *step {
	return Step(
		StepNameAptInstall,
		doAptInstall,
		WithAfter(StepNameAptUpdate),
	)
}

var pendingPackages = NewKey[map[string]struct{}]("pending-apt-packages")

func doAptInstall(ctx *Context) error {
	pkgSet, _ := Get(ctx, pendingPackages)
	if len(pkgSet) == 0 {
		return nil
	}
	cna := []string{"apt", "install", "--no-install-recommends"}
	for pkg := range pkgSet {
		cna = append(cna, pkg)
	}
	if err := Shell(
		ctx,
		cna,
		WithSudo(fmt.Sprintf("install %d packages", len(pkgSet))),
		// installation may prompt for things
		WithPassStdio(),
	); err != nil {
		return err
	}
	// clear the pending package list so that a little trickery can install more
	// package groups later, e.g. in case setting up some apt source requires
	// installing some packages.
	clear(pkgSet)
	return nil
}

func AddAptPackages(ctx *Context, names ...string) {
	pkgSet, _ := Get(ctx, pendingPackages)
	if pkgSet == nil {
		pkgSet = map[string]struct{}{}
		Save(ctx, pendingPackages, pkgSet)
	}
	for _, name := range names {
		pkgSet[name] = struct{}{}
	}
}

func init() {
	defaultStepFactories[StepNameAptUpdate] = aptUpdate
	defaultStepFactories[StepNameAptInstall] = aptInstall
}
