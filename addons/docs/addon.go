package docs

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/instance"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "docs",
		Description: func() string {
			return "Docs addon for including a docs-generator command"
		},
		// Initialize: initialize, // initialized below to avoid circular dependency
	},
	Config: config{},
}

func init() {
	addon.Definition.Initialize = initialize
}

type config struct{}

type option func(*config)

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	addon.RegisterIfNeeded()
}

func initialize() error {
	var manPath string
	var markdownPath string
	cmd := &cobra.Command{
		Use:    "docs",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ok := false

			root := cmd
			for {
				if p := root.Parent(); p != nil && p != root {
					root = p
				} else {
					break
				}
			}

			if manPath != "" {
				fmt.Println("Generating man pages ...")
				if err := doc.GenManTree(root, nil, manPath); err != nil {
					return err
				}
				ok = true
			}

			if markdownPath != "" {
				fmt.Println("Generating markdown documentation ...")
				if err := doc.GenMarkdownTree(root, markdownPath); err != nil {
					return err
				}
				ok = true
			}

			if !ok {
				return fmt.Errorf("must specify at least one type of documentation to generate")
			}

			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&manPath, "man", "", "Generate man pages")
	f.StringVar(&markdownPath, "markdown", "", "Generate markdown documentation")
	instance.AddCommands(cmd)
	return nil
}
