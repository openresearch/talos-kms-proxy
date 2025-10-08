package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v3"

	"github.openresearch.com/talos-kms-proxy/internal/acme"
	"github.openresearch.com/talos-kms-proxy/internal/kms"
)

// run executes the main routine and listens for incoming requests
func run(ctx context.Context, cmd *cli.Command) error {

	log.Info().Msgf("%s %s (%s), built %s, %s",
		appname,
		version,
		commit,
		date,
		runtime.Version(),
	)

	// create a channel to pass certificates on
	certsChannel := make(chan map[string][]byte)

	// create new suture service supervisor
	supervisor := suture.NewSimple(appname)

	// create new acme service instance
	a := acme.New(
		cmd.StringSlice("domain"),
		cmd.String("email"),
		cmd.String("workdir"),
		cmd.Bool("debug-mode"),
		certsChannel,
	)
	supervisor.Add(a)

	// create new kms server instance
	ks, err := kms.NewServer(
		cmd.String("listen-port"),
		cmd.String("workdir"),
		certsChannel,
	)
	if err != nil {
		return err
	}
	supervisor.Add(ks)

	// start services
	if err := supervisor.Serve(ctx); err != nil {
		return fmt.Errorf("supervisor: %w", err)
	}

	return nil
}

func prepare(ctx context.Context, cmd *cli.Command) (context.Context, error) {

	// configure logging
	logLevel := cmd.String("log-level")

	switch logLevel {
	case "TRACE", "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "DEBUG", "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "ERROR", "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "INFO", "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	// create working directory
	if err := os.MkdirAll(cmd.String("workdir"), 0750); err != nil {
		log.Error().Err(err).Msg("")
	}

	return ctx, nil
}
