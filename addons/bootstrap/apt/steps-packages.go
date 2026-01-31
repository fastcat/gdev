package apt

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/lib/shx"
)

// Name of the step registered by [AddAptUpdate]. Steps that modify apt sources
// should reference this as a `before` constraint.
const StepNameUpdate = "apt update"

// WithExtraUpdate adds a secondary `apt update` step with the given name. It
// will always run after the main `apt update` step. You may pass additional
// ordering constraints in the options.
//
// This is equivalent to calling [bootstrap.WithSteps] with the result of [ExtraUpdateStep].
func WithExtraUpdate(name string, opts ...bootstrap.StepOpt) bootstrap.Option {
	return bootstrap.WithSteps(ExtraUpdateStep(name, opts...))
}

// ExtraUpdateStep creates a secondary `apt update` step with the given name. It
// will always run after the main `apt update` step. You should pass additional
// ordering constraints in the options.
func ExtraUpdateStep(name string, opts ...bootstrap.StepOpt) *bootstrap.Step {
	opts = append([]bootstrap.StepOpt{bootstrap.AfterSteps(StepNameUpdate)}, opts...)
	return bootstrap.NewStep(name, DoUpdate, opts...)
}

func updateStep() *bootstrap.Step {
	return bootstrap.NewStep(StepNameUpdate, DoUpdate)
}

var sourcesDirty = bootstrap.NewKey[bool]("apt sources dirty")

func DoUpdate(ctx *bootstrap.Context) error {
	dirty, ok := bootstrap.Get(ctx, sourcesDirty)
	// TODO: heuristic if we can skip the update entirely, e.g. if no sources were
	// changed and it ran within the last hour or something?
	if ok && !dirty {
		// we ran apt update once before, nothing has changed since, skip it
		return nil
	}
	if _, err := shx.Run(
		ctx,
		// printing the list of sources being fetched is not interesting, need two
		// `-q` to achieve that
		[]string{"apt", "update", "-qq"},
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
//
// This is equivalent to calling [bootstrap.WithSteps] with the result of
// [ExtraInstallStep].
func WithExtraInstall(name string, opts ...bootstrap.StepOpt) bootstrap.Option {
	return bootstrap.WithSteps(ExtraInstallStep(name, opts...))
}

// ExtraInstallStep creates a secondary `apt install` step with the given name.
// It will always run after the main `apt install` step. You should pass
// additional ordering constraints in the options, e.g. ensuring this runs after
// a custom update and package selection steps.
func ExtraInstallStep(name string, opts ...bootstrap.StepOpt) *bootstrap.Step {
	opts = append(
		[]bootstrap.StepOpt{
			bootstrap.AfterSteps(StepNameInstall),
			bootstrap.SimFunc(simInstall),
		},
		opts...,
	)
	return bootstrap.NewStep(name, doInstall, opts...)
}

var pendingPackages = bootstrap.NewKey[map[string]struct{}]("pending-apt-packages")

func doInstall(ctx *bootstrap.Context) error {
	return DoInstall(ctx, []string{"--no-install-recommends"}, nil, "")
}

// DoInstall runs `apt install -y ...` with:
//
//   - Any extra options you pass. Including `--no-install-recommends` is often
//     a good idea
//   - All the packages registered as pending installation
//   - Any extra packages you pass
//
// After installation, the pending package set is cleared, and if the list of
// installed packages changed, the needs-reboot flag is set.
//
// If sudoPrompt is set, it will be used as the prompt for the sudo password.
// Otherwise a string noting the number of packages to be installed will be
// generated.
func DoInstall(
	ctx *bootstrap.Context,
	extraOpts []string,
	extraPackages []string,
	sudoPrompt string,
) error {
	pkgSet, _ := bootstrap.Get(ctx, pendingPackages)
	if pkgSet == nil {
		pkgSet = map[string]struct{}{}
		bootstrap.Save(ctx, pendingPackages, pkgSet)
	}
	if len(extraPackages) > 0 {
		// don't mutate the stored list
		pkgSet = maps.Clone(pkgSet)
		for _, pkg := range extraPackages {
			pkgSet[pkg] = struct{}{}
		}
	}
	if len(pkgSet) == 0 {
		return nil
	}

	// note the versions of target packages installedBefore before we start so we
	// can detect if things changed.
	installedBefore, err := DpkgInstalled(ctx)
	if err != nil {
		return err
	}

	cna := []string{"apt", "install", "--yes"}
	cna = append(cna, extraOpts...)
	offset := len(cna)
	for pkg := range pkgSet {
		cna = append(cna, pkg)
	}
	if sudoPrompt == "" {
		sudoPrompt = fmt.Sprintf("install %d packages", len(pkgSet))
	}
	// make printing deterministic
	slices.Sort(cna[3:])
	fmt.Printf("Installing: %s\n", strings.Join(cna[offset:], " "))
	if _, err := shx.Run(
		ctx,
		cna,
		shx.WithSudo(sudoPrompt),
		// installation may prompt for things
		shx.PassStdio(),
		shx.WithCombinedError(),
	); err != nil {
		return err
	}
	// clear the pending package list so that a little trickery can install more
	// package groups later, e.g. in case setting up some apt source requires
	// installing some packages.
	bootstrap.Clear(ctx, pendingPackages)

	// assume that installing or upgrading packages requires a reboot. Note that
	// we intentionally don't just look at the packages we were asked to install,
	// but the overall system in case dependencies changed.
	if installedAfter, err := DpkgInstalled(ctx); err != nil {
		return err
	} else if !maps.Equal(installedBefore, installedAfter) {
		bootstrap.SetNeedsReboot(ctx)
	}

	return nil
}

// InstallNeeded returns true if any queued packages or any extra listed are not
// already installed.
func InstallNeeded(
	ctx *bootstrap.Context,
	extras ...string,
) (bool, error) {
	pkgSet, _ := bootstrap.Get(ctx, pendingPackages)
	if pkgSet == nil {
		pkgSet = map[string]struct{}{}
		bootstrap.Save(ctx, pendingPackages, pkgSet)
	}
	if len(extras) > 0 {
		// don't mutate the stored list
		pkgSet = maps.Clone(pkgSet)
		for _, pkg := range extras {
			pkgSet[pkg] = struct{}{}
		}
	}
	return needsInstall(ctx, pkgSet)
}

func needsInstall(ctx *bootstrap.Context, set map[string]struct{}) (bool, error) {
	installed, err := DpkgInstalled(ctx)
	if err != nil {
		return true, err
	}
	for pkg := range set {
		if _, ok := installed[pkg]; !ok {
			return true, nil
		}
	}
	return false, nil
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

func AddExtraPackagesStep(
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
