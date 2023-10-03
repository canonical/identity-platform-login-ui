package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/canonical/identity-platform-login-ui/internal/version"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "version",
	Long:  "version prints the application version, bound to the git tag",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("app version: %s\n", version.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

}
