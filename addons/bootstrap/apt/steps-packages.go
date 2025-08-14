package apt

import (
	"fmt"
	"slices"
	"strings"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/shx"
)

// Name of the step registered by [AddAptUpdate]. Steps that modify apt sources
// should reference this as a `before` constraint.
const StepNameUpdate = "apt update"

// WithExtraUpdate adds a secondary `apt update` step with the given name. It
// will always run after the main `apt update` step. You may pass additional
// ordering constraints in the options.
func WithExtraUpdate(name string, opts ...bootstrap.StepOpt) bootstrap.Option {
	opts = append([]bootstrap.StepOpt{bootstrap.AfterSteps(StepNameUpdate)}, opts...)
	return bootstrap.WithSteps(bootstrap.NewStep(name, doUpdate, opts...))
}

func updateStep() *bootstrap.Step {
	return bootstrap.NewStep(StepNameUpdate, doUpdate)
}

var sourcesDirty = bootstrap.NewKey[bool]("apt sources dirty")

func doUpdate(ctx *bootstrap.Context) error {
	dirty, ok := bootstrap.Get(ctx, sourcesDirty)
	// TODO: heuristic if we can skip the update entirely, e.g. if no sources were
	// changed and it ran within the last hour or something?
	if ok && !dirty {
		// we ran apt update once before, nothing has changed since, skip it
		return nil
	}
	if _, err := shx.Run(
		ctx,
		[]string{"apt", "update"},
		shx.WithSudo("update available packages"),
		shx.PassStdio(),
	); err != nil {
		return err
	}
	bootstrap.Set(ctx, sourcesDirty, false)
	return nil
}

// ChangedSources will mark the apt sources list as dirty, so a secondary
// `apt update` step registered with [WithExtraUpdate] will actually run.
func ChangedSources(ctx *bootstrap.Context) {
	bootstrap.Set(ctx, sourcesDirty, true)
}

// Name of the step registered by [AddAptInstall]. This step will install
// pending packages enqueued with [AddPackages]. Set any step that uses that
// to be before this step.
const StepNameInstall = "apt install"

func installStep() *bootstrap.Step {
	return bootstrap.NewStep(
		StepNameInstall,
		doInstall,
		bootstrap.AfterSteps(StepNameUpdate),
		bootstrap.SimFunc(simInstall),
	)
}

// WithExtraInstall adds a secondary `apt install` step with the given name.
// It will always run after the main `apt install` step. You may pass additional
// ordering constraints in the options.
//
// You likely want to pair this with [WithExtraUpdate], one or more steps to
// add new apt sources that call [ChangedSources] and [AddPackages].
func WithExtraInstall(name string, opts ...bootstrap.StepOpt) bootstrap.Option {
	opts = append([]bootstrap.StepOpt{bootstrap.AfterSteps(StepNameInstall)}, opts...)
	return bootstrap.WithSteps(bootstrap.NewStep(name, doUpdate, opts...))
}

var pendingPackages = bootstrap.NewKey[map[string]struct{}]("pending-apt-packages")

func doInstall(ctx *bootstrap.Context) error {
	pkgSet, _ := bootstrap.Get(ctx, pendingPackages)
	if len(pkgSet) == 0 {
		return nil
	}
	cna := []string{"apt", "install", "--no-install-recommends", "--yes"}
	offset := len(cna)
	for pkg := range pkgSet {
		cna = append(cna, pkg)
	}
	// make printing deterministic
	slices.Sort(cna[3:])
	fmt.Printf("Installing: %s\n", strings.Join(cna[offset:], " "))
	if _, err := shx.Run(
		ctx,
		cna,
		shx.WithSudo(fmt.Sprintf("install %d packages", len(pkgSet))),
		// installation may prompt for things
		shx.PassStdio(),
		shx.WithCombinedError(),
	); err != nil {
		return err
	}
	// clear the pending package list so that a little trickery can install more
	// package groups later, e.g. in case setting up some apt source requires
	// installing some packages.
	clear(pkgSet)
	return nil
}

func simInstall(ctx *bootstrap.Context) error {
	pkgSet, _ := bootstrap.Get(ctx, pendingPackages)
	if len(pkgSet) == 0 {
		return nil
	}
	packages := make([]string, 0, len(pkgSet))
	for pkg := range pkgSet {
		packages = append(packages, pkg)
	}
	fmt.Printf("Would install: %s\n", strings.Join(packages, ", "))
	clear(pkgSet)
	return nil
}

// AddPackages adds the given package names to the pending list of packages
// to install. They will be installed by the next `apt install` step, either the
// "main" one, or one registered by [WithExtraInstall].
//
// The caller is responsible for ensuring that such a step runs after this.
func AddPackages(ctx *bootstrap.Context, names ...string) {
	pkgSet, _ := bootstrap.Get(ctx, pendingPackages)
	if pkgSet == nil {
		pkgSet = map[string]struct{}{}
		bootstrap.Save(ctx, pendingPackages, pkgSet)
	}
	added := []string{}
	for _, name := range names {
		if _, ok := pkgSet[name]; !ok {
			added = append(added, name)
			pkgSet[name] = struct{}{}
		}
	}
	if len(added) > 0 {
		fmt.Printf("Queued packages to install: %s\n", strings.Join(added, " "))
	}
}

func AddPackagesStep(
	stepName string,
	packages ...string,
) *bootstrap.Step {
	mark := func(ctx *bootstrap.Context) error {
		AddPackages(ctx, packages...)
		return nil
	}
	return bootstrap.NewStep(
		stepName,
		mark,
		// apt update will get added automatically
		bootstrap.BeforeSteps(StepNameInstall),
		// this just marks things in memory, so sim can be the same as run, so that
		// the sim apt install step shows the real list
		bootstrap.SimFunc(mark),
	)
}

// WithPackages is an option for [Configure] that will register a step to
// mark the given package(s) to be installed by the main `apt install` step.
func WithPackages(
	stepName string,
	packages ...string,
) bootstrap.Option {
	return bootstrap.WithSteps(AddPackagesStep(stepName, packages...))
}

func init() {
	bootstrap.WithDefaultStepFactory(StepNameUpdate, updateStep)
	bootstrap.WithDefaultStepFactory(StepNameInstall, installStep)
}
