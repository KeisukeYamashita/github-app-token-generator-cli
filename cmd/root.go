package cmd

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/KeisukeYamashita/github-app-token-generator-cli/cmd/version"
	"github.com/KeisukeYamashita/github-app-token-generator-cli/pkg/logging"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Execute(out io.Writer) error {
	return newRootCmd(out).Execute()
}

type rootOpt struct {
	retryMax int

	logFormat string
	logLevel  string
}

func newRootCmd(out io.Writer) *cobra.Command {
	opts := &rootOpt{}

	cmd := &cobra.Command{
		Use:     "github-app-token-generator-cli",
		Short:   "CLI to generate tokens for GitHub App",
		Args:    cobra.ExactArgs(3),
		Version: version.Version,
		RunE:    run(out, opts),
	}

	cmd.AddCommand(version.NewVersionCmd(out))
	cmd.PersistentFlags().IntVarP(&opts.retryMax, "retry", "r", 0, "retry count")
	cmd.PersistentFlags().StringVarP(&opts.logFormat, "log-format", "", "console", "format of the logs")
	cmd.PersistentFlags().StringVarP(&opts.logLevel, "log-level", "", "info", "output of the logs")

	return cmd
}

func run(out io.Writer, opts *rootOpt) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		log, err := logging.NewLogger(out, logging.Level(opts.logLevel), logging.Format(opts.logFormat))
		if err != nil {
			return err
		}

		appIntegrationIDstr, appInstallationIDstr, rsaPrivateKeyPemPath := args[0], args[1], args[2]
		appIntegrationID, err := strconv.ParseInt(appIntegrationIDstr, 0, 64)
		if err != nil {
			log.Fatal("INTEGRATION ID must be number", zap.String("integrationID", appIntegrationIDstr), zap.Error(err))
			return err
		}

		appInstallationID, err := strconv.ParseInt(appInstallationIDstr, 0, 64)
		if err != nil {
			log.Fatal("[INSTALLATION ID must be number", zap.String("installationID", appIntegrationIDstr), zap.Error(err))
			return err
		}

		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = opts.retryMax
		retryClient.Logger = nil
		c := retryClient.StandardClient()

		itr, err := ghinstallation.NewKeyFromFile(
			c.Transport,
			appIntegrationID,
			appInstallationID,
			rsaPrivateKeyPemPath,
		)

		if err != nil {
			log.Fatal("Failed to create new transport", zap.Error(err))
			return err
		}

		token, err := itr.Token(context.Background())
		if err != nil {
			log.Fatal("Failed to get token", zap.Error(err))
			return err
		}

		fmt.Printf("%s", token)
		return nil
	}
}
