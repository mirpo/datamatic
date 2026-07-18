package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/goforj/godump"
	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/logger"
	"github.com/mirpo/datamatic/runner"
	"github.com/mirpo/datamatic/utils"
	"github.com/rs/zerolog/log"
)

var (
	version string = "dev-build"
	commit  string = "commit"
)

func main() {
	// subcommand form: `datamatic validate -config x.yaml`
	validateOnly := len(os.Args) > 1 && os.Args[1] == "validate"
	args := os.Args[1:]
	if validateOnly {
		args = os.Args[2:]
	}

	cfg := config.NewConfig()

	var ver bool
	flag.BoolVar(&ver, "version", false, "Get current version of datamatic")
	flag.BoolVar(&cfg.Verbose, "verbose", cfg.Verbose, "Enable DEBUG logging level")
	flag.BoolVar(&cfg.LogPretty, "log-pretty", cfg.LogPretty, "Enable pretty logging, JSON when false")
	flag.StringVar(&cfg.ConfigFile, "config", cfg.ConfigFile, "Config file path")
	flag.StringVar(&cfg.OutputFolder, "output", cfg.OutputFolder, "Output folder path")
	flag.IntVar(&cfg.HTTPTimeout, "http-timeout", cfg.HTTPTimeout, "HTTP timeout: 0 - no timeout, if number - recommended to put high on poor hardware")
	flag.BoolVar(&cfg.ValidateResponse, "validate-response", cfg.ValidateResponse, "Validate JSON response from server to match the schema")

	flag.CommandLine.Parse(args) //nolint:errcheck // flag.ExitOnError exits on failure

	if ver {
		slog.Info("datamatic", "version", version, "commit", commit)
		return
	}

	loggerConfig := logger.LoggerConfig{
		Verbose:   cfg.Verbose,
		LogPretty: cfg.LogPretty,
	}
	logger.ConfigLogger(loggerConfig)

	if len(cfg.ConfigFile) == 0 {
		log.Fatal().Msg("Config path is required")
	}

	if err := utils.LoadConfigFile(cfg); err != nil {
		log.Fatal().Err(err).Msg("Config check failed")
	}

	if validateOnly {
		// command result, not a log event: stable stdout regardless of log settings
		fmt.Printf("Config is valid: %d steps\n", len(cfg.Steps))
		return
	}

	if cfg.Verbose {
		godump.Dump(cfg)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	r := runner.NewRunner(cfg)
	if err := r.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to execute runner")
	}
}
