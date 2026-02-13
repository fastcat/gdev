package cmd

import "github.com/spf13/cobra"

type FlagCompletionRegistrar func(string, cobra.CompletionFunc) error
