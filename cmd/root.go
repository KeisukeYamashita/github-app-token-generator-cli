package cmd

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/KeisukeYamashita/github-app-token-generator-cli/cmd/version"
	"github.com/KeisukeYamashita/github-app-token-generator-cli/pkg/logging"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// Ported from https://github.com/hashicorp/go-retryablehttp/blob/ff6d014e72d968e0f328637b209477ee09393175/client.go#L63-L71
var (
	// A regular expression to match the error returned by net/http when the
	// configured number of redirects is exhausted. This error isn't typed
	// specifically so we resort to matching on the error string.
	redirectsErrorRe = regexp.MustCompile(`stopped after \d+ redirects\z`)

	// A regular expression to match the error returned by net/http when the
	// scheme specified in the URL is invalid. This error isn't typed
	// specifically so we resort to matching on the error string.
	schemeErrorRe = regexp.MustCompile(`unsupported protocol scheme`)
)

func Execute(out io.Writer) error {
	return newRootCmd(out).Execute()
}

type rootOpt struct {
	retryMax   int
	timeout    string
	reqTimeout string

	logFormat string
	logLevel  string

	// For enterprise users
	url string
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
	cmd.PersistentFlags().IntVarP(&opts.retryMax, "retry", "r", 6, "retry count")
	cmd.PersistentFlags().StringVarP(&opts.timeout, "timeout", "t", "30s", "overall timeout")
	cmd.PersistentFlags().StringVar(&opts.reqTimeout, "request-timeout", "30s", "timeout for each request")
	cmd.PersistentFlags().StringVarP(&opts.logFormat, "log-format", "", "console", "format of the logs")
	cmd.PersistentFlags().StringVarP(&opts.logLevel, "log-level", "", "info", "output of the logs")
	cmd.PersistentFlags().StringVar(&opts.url, "url", "", "url of the GitHub API. Defaults to the https://api.github.com")

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

		reqTimeout, err := time.ParseDuration(opts.reqTimeout)
		if err != nil {
			log.Fatal("Can't parse timeout flag", zap.String("requestTimeout", opts.reqTimeout), zap.Error(err))
			return err
		}

		retryClient.HTTPClient.Timeout = reqTimeout
		retryClient.CheckRetry = checkRetry
		retryClient.RetryMax = opts.retryMax
		retryClient.Logger = nil
		c := retryClient.StandardClient()

		itr, err := ghinstallation.NewKeyFromFile(
			c.Transport,
			appIntegrationID,
			appInstallationID,
			rsaPrivateKeyPemPath,
		)
		if opts.url != "" {
			itr.BaseURL = opts.url
		}

		if err != nil {
			log.Fatal("Failed to create new transport", zap.Error(err))
			return err
		}

		timeout, err := time.ParseDuration(opts.timeout)
		if err != nil {
			log.Fatal("Can't parse timeout flag", zap.String("timeout", opts.timeout), zap.Error(err))
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		token, err := itr.Token(ctx)
		if err != nil {
			log.Fatal("Failed to get token", zap.Error(err))
			return err
		}

		fmt.Printf("%s", token)
		return nil
	}
}

// Inspired by https://github.com/hashicorp/go-retryablehttp/blob/ff6d014e72d968e0f328637b209477ee09393175/client.go#L411-L420
// Supported retry on timeout
func checkRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// do not retry on context.Canceled
	if err := ctx.Err(); err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("hoho")
			return false, ctx.Err()
		}
		fmt.Println("timeout")

		return true, nil
	}

	// don't propagate other errors
	shouldRetry, _ := baseRetryPolicy(resp, err)
	return shouldRetry, nil
}

// Ported from https://github.com/hashicorp/go-retryablehttp/blob/ff6d014e72d968e0f328637b209477ee09393175/client.go#L434-L473
func baseRetryPolicy(resp *http.Response, err error) (bool, error) {
	if err != nil {
		if v, ok := err.(*url.Error); ok {
			// Don't retry if the error was due to too many redirects.
			if redirectsErrorRe.MatchString(v.Error()) {
				return false, v
			}

			// Don't retry if the error was due to an invalid protocol scheme.
			if schemeErrorRe.MatchString(v.Error()) {
				return false, v
			}

			// Don't retry if the error was due to TLS cert verification failure.
			if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
				return false, v
			}
		}

		// The error is likely recoverable so retry.
		return true, nil
	}

	// 429 Too Many Requests is recoverable. Sometimes the server puts
	// a Retry-After response header to indicate when the server is
	// available to start processing request from client.
	if resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}

	// Check the response code. We retry on 500-range responses to allow
	// the server time to recover, as 500's are typically not permanent
	// errors and may relate to outages on the server side. This will catch
	// invalid response codes as well, like 0 and 999.
	if resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != 501) {
		return true, fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}

	return false, nil
}
