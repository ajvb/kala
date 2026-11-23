package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ajvb/kala/v2/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version, commit, build date and Go version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := fmt.Fprint(cmd.OutOrStdout(), version.Get())
		return err
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
	// "kala --version" prints the same as "kala version".
	RootCmd.Version = version.Get().Version
	RootCmd.SetVersionTemplate(version.Get().String())
}
