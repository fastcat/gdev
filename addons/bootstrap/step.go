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
	opts ...stepOpt,
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

type stepOpt func(*Step)

// WithSim sets the simulation function that will be run instead of just
// printing the step name in [Sim] (dry run) invocations.
func WithSim(f func(*Context) error) stepOpt {
	return func(s *Step) { s.sim = f }
}

// WithBefore adds reverse dependencies to the step
func WithBefore(names ...string) stepOpt {
	return func(s *Step) {
		for _, n := range names {
			s.before[n] = struct{}{}
		}
	}
}

// WithAfter adds normal dependencies to the step
func WithAfter(names ...string) stepOpt {
	return func(s *Step) {
		for _, n := range names {
			s.after[n] = struct{}{}
		}
	}
}
