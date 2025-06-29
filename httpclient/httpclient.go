package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
)

type HTTPError struct {
	StatusCode int
	Body       []byte
	Err        error
}

func (h *HTTPError) Error() string {
	return fmt.Sprintf("HTTP error %d: %v - body: %s", h.StatusCode, h.Err, string(h.Body))
}

func (h *HTTPError) NotFound() bool {
	return h.StatusCode == http.StatusNotFound
}

func (h *HTTPError) IsRetryable() bool {
	switch h.StatusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

func (h *HTTPError) IsPermanent() bool {
	switch h.StatusCode {
	case http.StatusBadRequest, // 400
		http.StatusUnauthorized,        // 401
		http.StatusForbidden,           // 403
		http.StatusNotFound,            // 404
		http.StatusMethodNotAllowed,    // 405
		http.StatusNotAcceptable,       // 406
		http.StatusConflict,            // 409
		http.StatusUnprocessableEntity: // 422
		return true
	default:
		return false
	}
}

func (h *HTTPError) IsRateLimited() bool {
	return h.StatusCode == http.StatusTooManyRequests
}

type Client struct {
	AuthToken  string
	BaseURL    string
	HTTPClient *http.Client
}

type Option func(*Client)

func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if c.HTTPClient == nil {
			c.HTTPClient = &http.Client{}
		}
		c.HTTPClient.Timeout = timeout
	}
}

func NewClient(baseURL string, authToken string, opts ...Option) *Client {
	c := &Client{
		BaseURL:   baseURL,
		AuthToken: authToken,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Client) Post(ctx context.Context, path string, requestBody interface{}, responseBody interface{}, headers http.Header) error {
	var jsonReqBody []byte
	var err error
	if requestBody != nil {
		jsonReqBody, err = json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("error marshaling request body: %w", err)
		}
	}

	fullURL, err := url.JoinPath(c.BaseURL, path)
	if err != nil {
		return fmt.Errorf("error joining URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewBuffer(jsonReqBody))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AuthToken)

	for key, values := range headers {
		for _, value := range values {
			req.Header.Set(key, value)
		}
	}

	log.Debug().Msgf("sending request to url: %s, req: %+v", fullURL, req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	log.Debug().Msgf("statusCode: %d, Body: %s\n", resp.StatusCode, string(respBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       respBytes,
			Err:        fmt.Errorf("unexpected status code: %d", resp.StatusCode),
		}
	}

	if responseBody != nil && len(respBytes) > 0 && resp.StatusCode != http.StatusNoContent {
		if err := json.Unmarshal(respBytes, responseBody); err != nil {
			return fmt.Errorf("error unmarshaling response body: %w, body: %s", err, string(respBytes))
		}
	}

	return nil
}
