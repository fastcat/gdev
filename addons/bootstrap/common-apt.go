package bootstrap

import (
	"fmt"
	"slices"
	"strings"

	"fastcat.org/go/gdev/shx"
)

// Name of the step registered by [AddAptUpdate]. Steps that modify apt sources
// should reference this as a `before` constraint.
const StepNameAptUpdate = "apt update"

// WithExtraAptUpdate adds a secondary `apt update` step with the given name. It
// will always run after the main `apt update` step. You may pass additional
// ordering constraints in the options.
func WithExtraAptUpdate(name string, opts ...StepOpt) option {
	opts = append([]StepOpt{AfterSteps(StepNameAptUpdate)}, opts...)
	return WithSteps(NewStep(name, doAptUpdate, opts...))
}

func aptUpdate() *Step {
	return NewStep(StepNameAptUpdate, doAptUpdate)
}

var sourcesDirty = NewKey[bool]("apt sources dirty")

func doAptUpdate(ctx *Context) error {
	dirty, ok := Get(ctx, sourcesDirty)
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
	Set(ctx, sourcesDirty, false)
	return nil
}

// ChangedAptSources will mark the apt sources list as dirty, so a secondary
// `apt update` step registered with [WithExtraAptUpdate] will actually run.
func ChangedAptSources(ctx *Context) {
	Set(ctx, sourcesDirty, true)
}

// Name of the step registered by [AddAptInstall]. This step will install
// pending packages enqueued with [AddAptPackages]. Set any step that uses that
// to be before this step.
const StepNameAptInstall = "apt install"

func aptInstall() *Step {
	return NewStep(
		StepNameAptInstall,
		doAptInstall,
		AfterSteps(StepNameAptUpdate),
		SimFunc(simAptInstall),
	)
}

// WithExtraAptInstall adds a secondary `apt install` step with the given name.
// It will always run after the main `apt install` step. You may pass additional
// ordering constraints in the options.
//
// You likely want to pair this with [WithExtraAptUpdate], one or more steps to
// add new apt sources that call [ChangedAptSources] and [AddAptPackages].
func WithExtraAptInstall(name string, opts ...StepOpt) option {
	opts = append([]StepOpt{AfterSteps(StepNameAptInstall)}, opts...)
	return WithSteps(NewStep(name, doAptUpdate, opts...))
}

var pendingPackages = NewKey[map[string]struct{}]("pending-apt-packages")

func doAptInstall(ctx *Context) error {
	pkgSet, _ := Get(ctx, pendingPackages)
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

func simAptInstall(ctx *Context) error {
	pkgSet, _ := Get(ctx, pendingPackages)
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

// AddAptPackages adds the given package names to the pending list of packages
// to install. They will be installed by the next `apt install` step, either the
// "main" one, or one registered by [WithExtraAptInstall].
//
// The caller is responsible for ensuring that such a step runs after this.
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

func AddAptPackagesStep(
	stepName string,
	packages ...string,
) *Step {
	mark := func(ctx *Context) error {
		AddAptPackages(ctx, packages...)
		return nil
	}
	return NewStep(
		stepName,
		mark,
		// apt update will get added automatically
		BeforeSteps(StepNameAptInstall),
		// this just marks things in memory, so sim can be the same as run, so that
		// the sim apt install step shows the real list
		SimFunc(mark),
	)
}

// WithAptPackages is an option for [Configure] that will register a step to
// mark the given package(s) to be installed by the main `apt install` step.
func WithAptPackages(
	stepName string,
	packages ...string,
) option {
	return WithSteps(AddAptPackagesStep(stepName, packages...))
}

func init() {
	defaultStepFactories[StepNameAptUpdate] = aptUpdate
	defaultStepFactories[StepNameAptInstall] = aptInstall
}
