package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

var (
	appname = "unknown"
	version = "unknown"
	date    = "unknown"
	commit  = "unknown"
)

// main is the execution entry point of the service
func main() {

	// create context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// run cli handler
	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Printf("%s %s (%s), built %s, %s\n",
			appname,
			version,
			commit,
			date,
			runtime.Version(),
		)
	}
	cmd := commands
	if err := cmd.Run(ctx, os.Args); err != nil {
		log.Fatal().Msg(err.Error())
		os.Exit(1)
	}
}
