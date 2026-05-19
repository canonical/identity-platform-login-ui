// Copyright 2026 Canonical Ltd
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/canonical/identity-platform-login-ui/internal/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get the application's version",
	Long:  `Get the application's version`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("App Version: %s\n", version.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
