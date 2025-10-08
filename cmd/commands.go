package main

import (
	"github.com/urfave/cli/v3"
)

var (
	commands = &cli.Command{
		Name:    appname,
		Usage:   "Talos KMS Server",
		Version: version,
		Before:  prepare,
		Action:  run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "listen-port",
				Usage:    "Service listen port",
				Value:    ":4050",
				Aliases:  []string{"p"},
				Sources:  cli.EnvVars("LISTEN_PORT"),
				Required: false,
			},
			&cli.StringFlag{
				Name:     "email",
				Usage:    "Email to use for ACME Client",
				Aliases:  []string{"e"},
				Sources:  cli.EnvVars("EMAIL"),
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:     "domain",
				Usage:    "Domain used in SAN filed for the server certificate (can be repeated)",
				Aliases:  []string{"d"},
				Sources:  cli.EnvVars("DOMAINS"),
				Required: true,
			},
			&cli.StringFlag{
				Name:     "workdir",
				Usage:    "Working directory to store files",
				Value:    ".taloskms",
				Aliases:  []string{"wd"},
				Sources:  cli.EnvVars("WORKDIR"),
				Required: false,
			},
			&cli.StringFlag{
				Name:     "log-level",
				Usage:    "Logging level to use",
				Value:    "info",
				Aliases:  []string{"l"},
				Sources:  cli.EnvVars("LOG_LEVEL"),
				Required: false,
			},
			&cli.StringFlag{
				Name:     "aws-kms-key-id",
				Usage:    "AWS KMS key ID",
				Sources:  cli.EnvVars("AWS_KMS_KEY_ID"),
				Required: true,
			},
			&cli.StringFlag{
				Name:     "aws-access-key-id",
				Usage:    "AWS access key ID",
				Sources:  cli.EnvVars("AWS_ACCESS_KEY_ID"),
				Required: true,
			},
			&cli.StringFlag{
				Name:     "aws-secret-access-key",
				Usage:    "AWS secret access key",
				Sources:  cli.EnvVars("AWS_SECRET_ACCESS_KEY"),
				Required: true,
			},
			&cli.StringFlag{
				Name:     "aws-hosted-zone-id",
				Usage:    "AWS hosted zone ID",
				Sources:  cli.EnvVars("AWS_HOSTED_ZONE_ID"),
				Required: true,
			},
			&cli.BoolFlag{
				Name:     "debug-mode",
				Usage:    "Run in debug mode (uses staging Let's Encrypt server)",
				Required: false,
				Sources:  cli.EnvVars("DEBUG_MODE"),
				Value:    false,
			},
		},
	}
)
