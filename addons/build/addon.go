package build

import (
	"fmt"
	"maps"
	"slices"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/stack"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "build",
		Description: func() string {
			return "Support for building repos/packages from source"
		},
		// Initialize: initialize,
	},
	Config: config{
		strategies: make(map[string]strategy),
	},
}

func init() {
	addon.Definition.Initialize = initialize
}

type config struct {
	strategies    map[string]strategy
	strategyOrder []string
}

type option func(*config)

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	addon.RegisterIfNeeded()
}

func WithStrategy(
	name string,
	detector Detector,
	supersedes []string,
) option {
	return func(c *config) {
		if _, ok := c.strategies[name]; ok {
			panic(fmt.Errorf("strategy %q already exists", name))
		}
		c.strategies[name] = strategy{
			name:       name,
			detector:   detector,
			supersedes: supersedes,
		}
	}
}

func initialize() error {
	// resolve strategy order, make sure there are no cycles
	if err := addon.Config.resolveStrategyOrder(); err != nil {
		return err
	}

	instance.AddCommandBuilders(makeCmd)

	stack.AddPreStartHookType[buildBeforeStart]()

	return nil
}

func (c *config) resolveStrategyOrder() error {
	// topological sort of strategies based on supersedes. We want strategies to
	// come before their supersedes, so we can take the first match in a simple
	// search. This means we need to convert supersedes to superseded-by first.
	supersededBy := make(map[string][]string, len(c.strategies))
	for name, s := range c.strategies {
		for _, sup := range s.supersedes {
			supersededBy[sup] = append(supersededBy[sup], name)
		}
	}

	c.strategyOrder = make([]string, 0, len(c.strategies))
	seen := make(map[string]bool)
	var visit func(name string, path []string) error
	visit = func(name string, path []string) error {
		if slices.Contains(path, name) {
			return fmt.Errorf("cycle detected in strategy order: %v", append(path, name))
		}
		if seen[name] {
			return nil
		}
		seen[name] = true

		subPath := append(path, name)
		// ensure deterministic order
		slices.Sort(supersededBy[name])
		for _, s := range supersededBy[name] {
			if err := visit(s, subPath); err != nil {
				return fmt.Errorf("%s.supersedes: %w", name, err)
			}
		}

		c.strategyOrder = append(c.strategyOrder, name)
		return nil
	}

	// start with a sorted list so the final order is deterministic
	for _, name := range slices.Sorted(maps.Keys(c.strategies)) {
		if err := visit(name, nil); err != nil {
			return fmt.Errorf("error resolving strategy order: %w", err)
		}
	}

	return nil
}
