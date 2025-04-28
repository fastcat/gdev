package bootstrap

type step struct {
	name   string
	run    func(*Context) error
	sim    func(*Context) error
	after  map[string]struct{}
	before map[string]struct{}
}

func Step(
	name string,
	run func(*Context) error,
	opts ...stepOpt,
) *step {
	s := &step{
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

type stepOpt func(*step)

// WithSim sets the simulation function that will be run instead of just
// printing the step name in [Sim] (dry run) invocations.
func WithSim(f func(*Context) error) stepOpt {
	return func(s *step) { s.sim = f }
}

// WithBefore adds reverse dependencies to the step
func WithBefore(names ...string) stepOpt {
	return func(s *step) {
		for _, n := range names {
			s.before[n] = struct{}{}
		}
	}
}

// WithAfter adds normal dependencies to the step
func WithAfter(names ...string) stepOpt {
	return func(s *step) {
		for _, n := range names {
			s.after[n] = struct{}{}
		}
	}
}
