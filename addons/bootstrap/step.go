package bootstrap

import "fastcat.org/go/gdev/internal"

type Step struct {
	name   string
	run    func(*Context) error
	sim    func(*Context) error
	after  map[string]struct{}
	before map[string]struct{}
	_      internal.NoCopy
}

// NewStep creates a new bootstrap step with the given name and run function.
//
// Dependencies, simulation (dry-run) mode special case, and other options can
// be set via the additional option arguments.
func NewStep(
	name string,
	run func(*Context) error,
	opts ...StepOpt,
) *Step {
	s := &Step{
		name:   name,
		run:    run,
		before: map[string]struct{}{},
		after:  map[string]struct{}{},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

type StepOpt func(*Step)

func (s *Step) With(opts ...StepOpt) *Step {
	for _, o := range opts {
		o(s)
	}
	return s
}

// SimFunc sets the simulation function that will be run instead of just
// printing the step name in [Sim] (dry run) invocations.
func SimFunc(f func(*Context) error) StepOpt {
	return func(s *Step) { s.sim = f }
}

// BeforeSteps adds reverse dependencies to the step
func BeforeSteps(names ...string) StepOpt {
	return func(s *Step) {
		for _, n := range names {
			s.before[n] = struct{}{}
		}
	}
}

// AfterSteps adds normal dependencies to the step
func AfterSteps(names ...string) StepOpt {
	return func(s *Step) {
		for _, n := range names {
			s.after[n] = struct{}{}
		}
	}
}
