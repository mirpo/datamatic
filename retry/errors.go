package retry

import (
	"errors"
	"net"

	"github.com/rs/zerolog/log"
)

type HTTPError interface {
	HTTPStatusCode() int
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	log.Debug().Msgf("Error: %v", err)

	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		statusCode := httpErr.HTTPStatusCode()
		if isRetryableStatusCode(statusCode) {
			log.Error().Msgf("Retryable HTTP error (status %d): %s", statusCode, err.Error())
			return true
		}
		log.Error().Msgf("Permanent HTTP error (status %d): %s", statusCode, err.Error())
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		log.Error().Msgf("Retryable network timeout: %s", err.Error())
		return true
	}

	return false
}

func isRetryableStatusCode(code int) bool {
	switch code {
	case 429, 500, 502, 503, 504: // Rate limit, server errors
		return true
	case 401, 403, 400, 404, 422: // Auth, client errors
		return false
	default:
		return false
	}
}
