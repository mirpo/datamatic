package retry

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
)

type Config struct {
	Enabled           bool          `yaml:"enabled"`
	MaxAttempts       int           `yaml:"maxAttempts"`
	InitialDelay      time.Duration `yaml:"initialDelay"`
	MaxDelay          time.Duration `yaml:"maxDelay"`
	BackoffMultiplier float64       `yaml:"backoffMultiplier"`
}

func NewDefaultConfig() Config {
	return Config{
		MaxAttempts:       3,
		InitialDelay:      1 * time.Second,
		MaxDelay:          10 * time.Second,
		BackoffMultiplier: 2.0,
		Enabled:           true,
	}
}

type ErrorClassifier func(error) bool

func Do(ctx context.Context, cfg Config, fn func() error, shouldRetry ErrorClassifier) error {
	if !cfg.Enabled {
		return fn()
	}

	retryFn := func() error {
		err := fn()
		if err == nil {
			return nil
		}

		if shouldRetry(err) {
			log.Warn().Err(err).Msg("Operation failed, will retry")
			return err
		}

		log.Error().Err(err).Msg("Operation failed with permanent error")
		return retry.Unrecoverable(err)
	}

	return retry.Do(
		retryFn,
		retry.Attempts(uint(cfg.MaxAttempts)),
		retry.Delay(cfg.InitialDelay),
		retry.MaxDelay(cfg.MaxDelay),
		retry.DelayType(retry.BackOffDelay),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			log.Info().Msgf("Retry attempt %d/%d after error: %v", n+1, cfg.MaxAttempts, err)
		}),
	)
}

func ShouldRetryHTTPError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *openai.RequestError
	log.Debug().Msgf("HTTP error: %v", err)
	if errors.As(err, &apiErr) {
		switch apiErr.HTTPStatusCode {
		case 401, 403:
			log.Error().Msgf("Permanent API error (status %d): %s", apiErr.HTTPStatusCode, apiErr.Error())
			return false
		case 429, 500, 502, 503, 504:
			log.Error().Msgf("Retryable API error (status %d): %s", apiErr.HTTPStatusCode, apiErr.Error())
			return true
		case 400, 404, 422:
			log.Error().Msgf("Permanent API error (status %d): %s", apiErr.HTTPStatusCode, apiErr.Error())
			return false
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}
