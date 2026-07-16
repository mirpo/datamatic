// Package llmtest provides a mock OpenAI-compatible chat-completions server for tests.
package llmtest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type Server struct {
	URL   string
	Delay time.Duration // set before first request; simulates a slow server

	server    *httptest.Server
	mu        sync.Mutex
	responses []string
	requests  []map[string]interface{}
}

// NewServer returns a mock chat-completions server that answers with the given
// message contents in order; the last response repeats for extra calls.
func NewServer(t *testing.T, responses ...string) *Server {
	t.Helper()
	s := &Server{responses: responses}

	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.Delay > 0 {
			select {
			case <-time.After(s.Delay):
			case <-r.Context().Done():
				return // client gave up; don't hold server shutdown
			}
		}

		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		_ = json.Unmarshal(body, &req)

		s.mu.Lock()
		s.requests = append(s.requests, req)
		idx := len(s.requests) - 1
		if idx >= len(s.responses) {
			idx = len(s.responses) - 1
		}
		content := s.responses[idx]
		s.mu.Unlock()

		model, _ := req["model"].(string)
		resp := map[string]interface{}{
			"id":     "mock",
			"object": "chat.completion",
			"model":  model,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"finish_reason": "stop",
					"message":       map[string]interface{}{"role": "assistant", "content": content},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(s.server.Close)

	s.URL = s.server.URL
	return s
}

func (s *Server) CallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

// Requests returns the decoded JSON bodies of all received requests.
func (s *Server) Requests() []map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]map[string]interface{}(nil), s.requests...)
}
