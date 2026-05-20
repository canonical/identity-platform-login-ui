/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"net/url"

	authz "github.com/canonical/identity-platform-login-ui/internal/authorization"
	logging "github.com/canonical/identity-platform-login-ui/internal/logging"
	monitoring "github.com/canonical/identity-platform-login-ui/internal/monitoring"
	fga "github.com/canonical/identity-platform-login-ui/internal/openfga"
	tracing "github.com/canonical/identity-platform-login-ui/internal/tracing"
	"github.com/spf13/cobra"
)

// createFgaModelCmd represents the createFgaModel command
var createFgaModelCmd = &cobra.Command{
	Use:   "create-fga-model",
	Short: "Creates an openfga model",
	Long:  `Creates an openfga model`,
	Run: func(cmd *cobra.Command, args []string) {
		apiUrl, _ := cmd.Flags().GetString("fga-api-url")
		apiToken, _ := cmd.Flags().GetString("fga-api-token")
		storeId, _ := cmd.Flags().GetString("store-id")
		createModel(apiUrl, apiToken, storeId)
	},
}

func init() {
	rootCmd.AddCommand(createFgaModelCmd)

	createFgaModelCmd.PersistentFlags().String("fga-api-url", "", "The openfga API URL")
	createFgaModelCmd.PersistentFlags().String("fga-api-token", "", "The openfga API token")
	createFgaModelCmd.PersistentFlags().String("store-id", "", "The openfga store to create the model in")
	createFgaModelCmd.MarkPersistentFlagRequired("api-url")
	createFgaModelCmd.MarkPersistentFlagRequired("api-token")
	createFgaModelCmd.MarkPersistentFlagRequired("store-id")
}

func createModel(apiUrl, apiToken, storeId string) {
	logger := logging.NewNoopLogger()
	tracer := tracing.NewNoopTracer()
	monitor := monitoring.NewNoopMonitor("", logger)
	scheme, host, err := parseURL(apiUrl)
	if err != nil {
		panic(err)
	}
	cfg := fga.NewConfig(scheme, host, storeId, apiToken, "", false, tracer, monitor, logger)
	fgaClient := fga.NewClient(cfg)
	modelId, err := fgaClient.WriteModel(context.Background(), []byte(authz.AuthModel))
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created model: %s\n", modelId)
}

func parseURL(s string) (string, string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", "", err
	}
	return u.Scheme, u.Host, nil
}
