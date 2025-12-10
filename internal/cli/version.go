package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/animus-coder/animus-coder/internal/version"
)

// NewVersionCmd prints the compiled version details.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show mycodex version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version.Full())
		},
	}
}
