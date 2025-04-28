package bootstrap

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
)

type plan struct {
	// steps to run in an order that will satisfy their dependencies.
	ordered []*step
	// steps whose dependencies haven't been registered yet and thus can't be
	// placed in the ordered list.
	pending []*step
}

var defaultPlan plan

func (p *plan) addSteps(steps ...*step) {
	p.pending = append(p.pending, steps...)
	p.tryResolveAll()
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
	unsatisfied := maps.Clone(s.dependencies)
	for _, s2 := range p.ordered {
		delete(unsatisfied, s2.name)
	}
	if len(unsatisfied) != 0 {
		return false
	}
	p.ordered = append(p.ordered, s)
	return true
}

func AddStep(s *step) {
	defaultPlan.addSteps(s)
}

func (p *plan) ready() bool {
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
	if !p.ready() {
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
