package build

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func makeCmd() *cobra.Command {
	var opts Options
	var strategy string
	buildCmd := &cobra.Command{
		Use:   "build <dirs...>",
		Args:  cobra.MinimumNArgs(1),
		Short: "Build a project or directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			var builders []Builder
			var strategies []string
			for _, arg := range args {
				if !strings.HasPrefix(arg, ".") {
					return fmt.Errorf("only relative paths are supported, got %q", arg)
				}
				root, err := filepath.Abs(arg)
				if err != nil {
					return fmt.Errorf("error getting absolute path for %q: %w", arg, err)
				}
				var b Builder
				sn := strategy
				if sn != "" {
					b, err = TryStrategy(root, sn)
					if err != nil {
						return fmt.Errorf("error trying strategy %q for %q: %w", sn, root, err)
					} else if b == nil {
						return fmt.Errorf("strategy %q is not supported for %q", sn, root)
					}
				} else {
					sn, b, err = DetectStrategy(root)
					if err != nil {
						return fmt.Errorf("error detecting build strategy for %q: %w", root, err)
					} else if b == nil {
						return fmt.Errorf("no build strategy found for %q", root)
					}
				}
				builders = append(builders, b)
				strategies = append(strategies, sn)
			}
			// TODO: support concurrent builds
			for i, b := range builders {
				if opts.Verbose {
					fmt.Printf("Building %s with %s\n", args[i], strategies[i])
				}
				if err := b.BuildAll(cmd.Context(), opts); err != nil {
					return fmt.Errorf("error building %s: %w", args[i], err)
				}
			}
			return nil
		},
	}
	buildCmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "print verbose output")
	buildCmd.Flags().StringVar(&strategy, "strategy", "",
		"use a specific build strategy (default: auto-detect)")

	return buildCmd
}
