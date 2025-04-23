package instance

import "github.com/spf13/cobra"

// Commands is a list of functions to run during app init to add additional
// commands to the Root command. They will be called from
// [fastcat.org/go/gdev/cmd/Root] during app startup.
var Commands []func() *cobra.Command
