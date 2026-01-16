package bootstrap

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
)

type Plan struct {
	byName map[string]*Step
	// steps to run in an order that will satisfy their dependencies.
	ordered []*Step
	// steps whose dependencies haven't been registered yet and thus can't be
	// placed in the ordered list.
	pending []*Step
}

func NewPlan() *Plan {
	return &Plan{byName: map[string]*Step{}}
}

func (p *Plan) AddSteps(steps ...*Step) {
	for _, s := range steps {
		if p.byName[s.name] != nil {
			panic(fmt.Errorf("already have step named %s", s.name))
		}
		if len(s.before) != 0 && len(p.ordered) != 0 {
			panic(fmt.Errorf(
				"cannot add step %s with non-empty before list once deps already resolved",
				s.name,
			))
		}
		p.byName[s.name] = s
		p.pending = append(p.pending, s)
	}
}

func (p *Plan) tryResolveAll() {
	progress := true
	for progress {
		progress = false
		for i, s := range p.pending {
			if p.resolveOne(s) {
				progress = true
				p.pending = slices.Delete(p.pending, i, i+1)
				break
			}
		}
	}
}

func (p *Plan) resolveOne(s *Step) bool {
	unsatisfied := maps.Clone(s.after)
	for _, s2 := range p.ordered {
		delete(unsatisfied, s2.name)
	}
	if len(unsatisfied) != 0 {
		// this step has an after constraint on a step that isn't already in the
		// ordered list
		return false
	}
	if slices.ContainsFunc(p.pending, func(s2 *Step) bool {
		_, ok := s2.before[s.name]
		return ok
	}) {
		// some other step still in the pending list has a before constraint on this
		// one
		return false
	}
	p.ordered = append(p.ordered, s)
	return true
}

// Ready tries to resolve all dependencies between plan steps to form a
// well-ordered plan. It returns whether this was successful.
//
// You will usually want to call [*Plan.AddDefaultSteps] before this, or
// alternately do so if this fails before trying again.
func (p *Plan) Ready() bool {
	p.tryResolveAll()
	return len(p.pending) == 0
}

func (p *Plan) Run(ctx context.Context) error {
	bc, err := p.prepare(ctx)
	if err != nil {
		return err
	}

	for _, s := range p.ordered {
		fmt.Printf("Running %s ...\n", s.name)
		if err := s.run(bc); err != nil {
			return err
		}
	}

	fmt.Println()
	fmt.Println("Bootstrap completed successfully")

	if needsReboot(bc) {
		fmt.Println()
		fmt.Println("IMPORTANT: You need to reboot before you can use the newly installed/updated tools!")
		fmt.Println()
	}

	return nil
}

func (p *Plan) prepare(ctx context.Context) (*Context, error) {
	bc, ok := ctx.(*Context)
	if !ok {
		bc = NewContextWithDefaults(ctx)
	}
	if !p.Ready() {
		names := make([]string, 0, len(p.pending))
		for _, s := range p.pending {
			names = append(names, s.name)
		}
		if !p.debugCircularDeps() {
			p.debugMissingDeps()
		}
		return nil, fmt.Errorf("plan has unresolved dependencies blocking %s", strings.Join(names, ", "))
	}
	if len(p.ordered) == 0 {
		return bc, fmt.Errorf("no bootstrap actions registered")
	}

	return bc, nil
}

func (p *Plan) debugCircularDeps() bool {
	isOrdered := make(map[string]bool, len(p.ordered))
	for _, s := range p.ordered {
		isOrdered[s.name] = true
	}
	afterByName := make(map[string]map[string]struct{}, len(p.pending))
	for _, s := range p.pending {
		m := maps.Clone(s.after)
		// trim out satisfied deps
		for n := range m {
			if _, ok := isOrdered[n]; ok {
				delete(m, n)
			}
		}
		afterByName[s.name] = m
	}
	for _, s := range p.pending {
		for beforeName := range s.before {
			if _, ok := isOrdered[beforeName]; !ok {
				if afterByName[beforeName] == nil {
					afterByName[beforeName] = make(map[string]struct{})
				}
				afterByName[beforeName][s.name] = struct{}{}
			}
		}
	}
	// find cycles
	visited := make(map[string]bool, len(afterByName))
	var visit func(name string, stack []string) bool
	visit = func(name string, stack []string) bool {
		if visited[name] {
			return false
		}
		visited[name] = true
		stack = append(stack, name)
		for depName := range afterByName[name] {
			for i, sn := range stack {
				if sn == depName {
					fmt.Printf("circular dependency: %s\n", strings.Join(append(stack[i:], depName), " -> "))
					return true
				}
			}
			if visit(depName, stack) {
				return true
			}
		}
		return false
	}
	found := false
	for name := range afterByName {
		if !visited[name] {
			// don't stop after one cycle, find them all
			if visit(name, nil) {
				found = true
			}
		}
	}
	return found
}

func (p *Plan) debugMissingDeps() {
	isOrdered := make(map[string]bool, len(p.ordered))
	for _, s := range p.ordered {
		isOrdered[s.name] = true
	}
	for _, s := range p.pending {
		m := maps.Clone(s.after)
		// trim out satisfied deps
		for n := range m {
			if _, ok := isOrdered[n]; ok {
				delete(m, n)
			}
		}
		if len(m) == 0 {
			continue
		}
		missing := slices.Sorted(maps.Keys(m))
		fmt.Printf("step %s is missing dependencies: %s\n", s.name, strings.Join(missing, ", "))
	}
}

func (p *Plan) Sim(ctx context.Context) error {
	bc, err := p.prepare(ctx)
	if err != nil {
		return err
	}

	for _, s := range p.ordered {
		if s.sim == nil {
			fmt.Printf("Would run %s\n", s.name)
			continue
		}
		fmt.Printf("Simulating %s ...\n", s.name)
		if err := s.sim(bc); err != nil {
			return err
		}
	}

	fmt.Println()
	fmt.Println("Bootstrap simulated successfully")

	return nil
}

// Add any of the known "Default" steps that are referenced in existing step
// dependencies but not already added to the plan.
func (p *Plan) AddDefaultSteps() {
	fill := func(names map[string]struct{}) bool {
		changed := false
		for name := range names {
			if p.byName[name] != nil {
				continue
			} else if f := defaultStepFactories[name]; f != nil {
				p.AddSteps(f())
				changed = true
			}
		}
		return changed
	}
	// have to loop this until we make no changes, because a step we add may
	// itself need more steps added
	todo := true
	for todo {
		todo = false
		// already ordered dependencies may have "before" links that need to be filled in
		for _, s := range p.ordered {
			if fill(s.before) {
				todo = true
			}
		}
		// pending steps may need defaults filled in both positions
		for _, s := range p.pending {
			if fill(s.before) {
				todo = true
			}
			if fill(s.after) {
				todo = true
			}
		}
	}
}

var defaultStepFactories = map[string]func() *Step{}

// WithDefaultStepFactory registers a function that will be called to create a
// default step with the given name. This is used to register steps that are
// referenced in existing step dependencies but not already added to the plan.
//
// These steps essentially will be run if and only if some other step depends on
// them.
//
// Normally this should only be used by bootstrap internals.
func WithDefaultStepFactory(name string, f func() *Step) {
	addon.CheckNotInitialized()
	if _, ok := defaultStepFactories[name]; ok {
		panic(fmt.Errorf("default step factory %s already registered", name))
	}
	defaultStepFactories[name] = f
}
