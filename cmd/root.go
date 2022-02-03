package cmd

import (
	"github.com/KeisukeYamashita/github-app-token-generator-cli/cmd/version"
	"github.com/spf13/cobra"
)

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "github-app-token-generator-cli",
		Short:   "CLI to generate tokens for GitHub App",
		Version: version.Version,
		RunE:    run,
	}

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	return nil
}
