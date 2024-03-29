# GitHub App Token Generator CLI

## Install

```console
$ go get -u github.com/KeisukeYamashita/github-app-token-generator-cli
```

## Usage

```console
$ github-app-token-generator-cli --help
CLI to generate tokens for GitHub App

Usage:
  github-app-token-generator-cli [flags]
  github-app-token-generator-cli [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version     

Flags:
  -h, --help                     help for github-app-token-generator-cli
      --log-format string        format of the logs (default "console")
      --log-level string         output of the logs (default "info")
      --request-timeout string   timeout for each request (default "30s")
  -r, --retry int                retry count (default 5)
  -t, --timeout string           overall timeout (default "1s")
      --url                      url of the GitHub API

Use "github-app-token-generator-cli [command] --help" for more information about a command.
```

## GitHub Enterprise

For users using GitHub enterprise, you can specify the GitHub API's URL with `--url` flag as below:

```console
$ github-app-token-generator-cli xxx yyy private-key.pem  --url https://<YOUR_HOST>/api/v3
```
