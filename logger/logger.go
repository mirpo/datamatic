package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type LoggerConfig struct {
	Verbose   bool
	LogPretty bool
}

func ConfigLogger(loggerConfig LoggerConfig) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if loggerConfig.Verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if loggerConfig.LogPretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}
