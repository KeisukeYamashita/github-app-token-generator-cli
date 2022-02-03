package cmd

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/KeisukeYamashita/github-app-token-generator-cli/cmd/version"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/spf13/cobra"
)

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "github-app-token-generator-cli",
		Short:   "CLI to generate tokens for GitHub App",
		Args:    cobra.ExactArgs(3),
		Version: version.Version,
		Run:     run,
	}

	return cmd
}

func run(cmd *cobra.Command, args []string) {
	appIntegrationIDstr, appInstallationIDstr, rsaPrivateKeyPemPath := args[1], args[2], args[3]
	appIntegrationID, err := strconv.ParseInt(appIntegrationIDstr, 0, 64)
	if err != nil {
		log.Fatalf("[ERROR] INTEGRATION ID must be number: %s", err)
	}

	appInstallationID, err := strconv.ParseInt(appInstallationIDstr, 0, 64)
	if err != nil {
		log.Fatalf("[ERROR] INSTALLATION ID must be number: %s", err)
	}

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	c := retryClient.StandardClient()

	itr, err := ghinstallation.NewKeyFromFile(
		c.Transport,
		appIntegrationID,
		appInstallationID,
		rsaPrivateKeyPemPath,
	)

	if err != nil {
		log.Fatalf("Failed to create new trasport: %s\n", err)
	}

	token, err := itr.Token(context.Background())
	if err != nil {
		log.Fatalf("Failed to get token: %s\n", err)
	}

	fmt.Printf("%s", token)
}
