package retry

import (
	"context"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/rs/zerolog/log"
)

type Config struct {
	MaxAttempts       int           `yaml:"maxAttempts"`
	InitialDelay      time.Duration `yaml:"initialDelay"`
	MaxDelay          time.Duration `yaml:"maxDelay"`
	BackoffMultiplier float64       `yaml:"backoffMultiplier"`
}

func Execute[T any](ctx context.Context, cfg Config, fn func() (T, error)) (T, error) {
	var result T

	retryFn := func() error {
		res, err := fn()
		if err == nil {
			result = res
			return nil
		}

		if IsRetryable(err) {
			log.Warn().Err(err).Msg("Operation failed, will retry")
			return err
		}

		log.Error().Err(err).Msg("Operation failed with permanent error")
		return retry.Unrecoverable(err)
	}

	err := retry.Do(
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

	return result, err
}
