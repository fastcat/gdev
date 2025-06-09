package build

import (
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_config_resolveStrategyOrder(t *testing.T) {
	tests := []struct {
		name       string
		supersedes map[string][]string
		wantOrder  []string
		assertion  assert.ErrorAssertionFunc
	}{
		{
			name: "no deps",
			supersedes: map[string][]string{
				"a": {},
				"b": {},
				"c": {},
			},
			wantOrder: []string{"a", "b", "c"},
			assertion: assert.NoError,
		},
		{
			name: "linear deps",
			supersedes: map[string][]string{
				"a": {},
				"b": {"a"},
				"c": {},
				"d": {"c"},
			},
			wantOrder: []string{"a", "b", "c", "d"},
			assertion: assert.NoError,
		},
		{
			name: "reverse linear deps",
			supersedes: map[string][]string{
				"d": {},
				"c": {"d"},
				"b": {},
				"a": {"b"},
			},
			wantOrder: []string{"b", "a", "d", "c"},
			assertion: assert.NoError,
		},
		{
			name: "deep deps",
			supersedes: map[string][]string{
				"a": {},
				"b": {"a"},
				"c": {"b"},
				"d": {"c"},
				"e": {"a", "b", "c"},
				"f": {"a", "b", "c", "d", "e"},
			},
			wantOrder: []string{"a", "b", "c", "d", "e", "f"},
			assertion: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{
				strategies: make(map[string]strategy, len(tt.supersedes)),
			}
			for n, sup := range tt.supersedes {
				c.strategies[n] = strategy{
					name:       n,
					supersedes: sup,
				}
			}
			err := c.resolveStrategyOrder()
			// if the error check fails, we know the populated order will be nonsense
			if tt.assertion(t, err) && err == nil {
				assert.Equal(t, tt.wantOrder, c.strategyOrder, "strategy order should be correct")
				// strategies should always come before their supersedes
				assertSuperseedsOrder(t, c)

				// fuzz a bit by scrambling all the names
				orig := slices.Collect(maps.Keys(c.strategies))
				// make a shuffled copy of orig
				// TODO: use random names instead
				newNames := slices.Clone(orig)
				rand.Shuffle(len(orig), func(i, j int) { newNames[i], newNames[j] = newNames[j], newNames[i] })
				renamed := make(map[string]string, len(c.strategies))
				for i, n := range orig {
					renamed[n] = newNames[i]
				}
				renameSlice := func(s []string) []string {
					ret := slices.Clone(s)
					for i, n := range ret {
						ret[i] = renamed[n]
					}
					return ret
				}
				c := &config{
					strategies: make(map[string]strategy, len(tt.supersedes)),
				}
				for n, sup := range tt.supersedes {
					c.strategies[renamed[n]] = strategy{
						name:       renamed[n],
						supersedes: renameSlice(sup),
					}
				}
				err := c.resolveStrategyOrder()
				// if the error check fails, we know the populated order will be nonsense
				if tt.assertion(t, err) && err == nil {
					// the final order may be different due to the randomization, so we
					// can't check against tt.wantOrder
					assertSuperseedsOrder(t, c)
				}
			}
		})
	}
}

func Fuzz_config_resolveStrategyOrder(f *testing.F) {
	// define a structure so any sequence of uint8 values is a valid test case.
	// the list is interpreted as sub-lists, captured as {l, [l]sups}. the name is
	// taken as 'a' + the index of the sub-list. l is taken mod 8 to limit the max
	// number of supersedes. sups are taken mod the total list size minus one,
	// skipping pointing at themselves. if the last sub-list doesn't have enough
	// elements, we truncate it. we take at most 26 sub-lists, so we can keep
	// things to single letter lower case.

	interp := func(t *testing.T, data []byte) *config {
		// we need to split the data into sub-lists before we can interpret it.
		inter := [][]byte{}
		for len(data) > 0 && len(inter) < 26 {
			// take the sups length mod 8
			l := int(data[0]) % 8
			if len(data) < 1+l {
				// not enough data for the sub-list, truncate
				l = len(data) - 1
			}
			inter = append(inter, data[1:1+l])
			data = data[1+l:]
		}
		// convert all the numbers to names
		c := &config{
			strategies: make(map[string]strategy, len(inter)),
		}
		for idx, item := range inter {
			name := string('a' + rune(idx%26))
			supersedes := make([]string, 0, len(item))
			if len(inter) > 1 {
				for _, sup := range item {
					supIdx := sup % byte(len(inter)-1) // skip self
					if int(supIdx) >= idx {
						supIdx++
					}
					supName := string('a' + rune(supIdx%26))
					assert.NotEqual(t, name, supName, "strategy cannot supersede itself")
					supersedes = append(supersedes, supName)
				}
			}
			c.strategies[name] = strategy{
				name:       name,
				supersedes: supersedes,
			}
		}

		return c
	}

	f.Add([]byte{})                    // no strategies
	f.Add([]byte{0, 0, 0, 0})          // 4 strategies, no supersedes
	f.Add([]byte{0, 1, 0, 1, 1, 1, 2}) // 4 strategies, linear stack

	f.Fuzz(func(t *testing.T, data []byte) {
		c := interp(t, data)
		{
			ss := make(map[string]string, len(c.strategies))
			for n, s := range c.strategies {
				sb := strings.Builder{}
				sb.Grow(len(s.supersedes))
				for _, sup := range s.supersedes {
					sb.WriteString(sup)
				}
				ss[n] = sb.String()
			}
			t.Logf("fuzzing with data: %v", ss)
		}
		err := c.resolveStrategyOrder()
		if err != nil {
			require.NotContains(t, err.Error(), "not found")
			t.Skipf("error resolving strategy order: %v", err)
		}
		assertSuperseedsOrder(t, c)
	})
}

func assertSuperseedsOrder(t *testing.T, c *config) {
	seen := make(map[string]bool, len(c.strategyOrder))
	for i, n := range c.strategyOrder {
		seen[n] = true
		for _, sup := range c.strategies[n].supersedes {
			assert.True(t, seen[sup],
				"strategy %q should come before its supersedes %q: %v",
				n,
				sup,
				c.strategyOrder[:i+1],
			)
		}
	}
}
