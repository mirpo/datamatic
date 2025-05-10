package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/gookit/goutil/dump"
	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/logger"
	"github.com/mirpo/datamatic/runner"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

var (
	version string = "dev-build"
	commit  string = "commit"
)

func main() {
	cfg := config.NewConfig()

	var ver bool
	flag.BoolVar(&ver, "version", false, "Get current version of datamatic")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Enable DEBUG logging level")
	flag.BoolVar(&cfg.LogPretty, "log-pretty", true, "Enable pretty logging, JSON when false")
	flag.StringVar(&cfg.ConfigFile, "config", "", "Config file path")
	flag.StringVar(&cfg.OutputFolder, "output", "dataset", "Output folder path")
	flag.IntVar(&cfg.HTTPTimeout, "http-timeout", 300, "HTTP timeout: 0 - no timeout, if number - recommended to put high on poor hardware")
	flag.BoolVar(&cfg.ValidateResponse, "validate-response", true, "Validate JSON response from server to match the schema")
	flag.BoolVar(&cfg.SkipCliWarning, "skip-cli-warning", false, "Skip external CLI warning")

	flag.Parse()

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

	yamlConfig, err := os.ReadFile(cfg.ConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Reading config file")
	}

	err = yaml.Unmarshal(yamlConfig, &cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Parsing config file")
	}

	err = cfg.Validate()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to validate config file")
	}

	if cfg.Verbose {
		dump.P(cfg)
	}

	r := runner.NewRunner(cfg)
	if err := r.Run(); err != nil {
		log.Fatal().Err(err).Msg("failed to execute runner")
		os.Exit(1)
	}
}
