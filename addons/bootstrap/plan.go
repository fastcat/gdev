package bootstrap

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
)

type plan struct {
	byName map[string]*step
	// steps to run in an order that will satisfy their dependencies.
	ordered []*step
	// steps whose dependencies haven't been registered yet and thus can't be
	// placed in the ordered list.
	pending []*step
}

func NewPlan() *plan {
	return &plan{byName: map[string]*step{}}
}

func (p *plan) AddSteps(steps ...*step) {
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

func (p *plan) tryResolveAll() {
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

func (p *plan) resolveOne(s *step) bool {
	unsatisfied := maps.Clone(s.after)
	for _, s2 := range p.ordered {
		delete(unsatisfied, s2.name)
	}
	if len(unsatisfied) != 0 {
		// this step has an after constraint on a step that isn't already in the
		// ordered list
		return false
	}
	if slices.ContainsFunc(p.pending, func(s2 *step) bool {
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

func (p *plan) Ready() bool {
	p.tryResolveAll()
	return len(p.pending) == 0
}

func (p *plan) Run(ctx context.Context) error {
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
	fmt.Println("Done")
	return nil
}

func (p *plan) prepare(ctx context.Context) (*Context, error) {
	bc, ok := ctx.(*Context)
	if !ok {
		bc = NewContext(ctx)
	}
	if !p.Ready() {
		names := make([]string, 0, len(p.pending))
		for _, s := range p.pending {
			names = append(names, s.name)
		}
		return nil, fmt.Errorf("plan has unresolved dependencies blocking %s", strings.Join(names, ", "))
	}
	if len(p.ordered) == 0 {
		return bc, fmt.Errorf("no bootstrap actions registered")
	}

	return bc, nil
}

func (p *plan) Sim(ctx context.Context) error {
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
	fmt.Println("Done")
	return nil
}

// Add any of the known "Default" steps that are referenced in existing step
// dependencies but not already added to the plan.
func (p *plan) AddDefaultSteps() {
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

var defaultStepFactories = map[string]func() *step{}
