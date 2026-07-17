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
	URL        string
	Delay      time.Duration // set before first request; simulates a slow server
	EchoPrompt bool          // when true, respond with the last user message instead of scripted content

	server    *httptest.Server
	mu        sync.Mutex
	responses []string
	requests  []map[string]interface{}
	inFlight  int
	maxInFlt  int
}

// NewServer returns a mock chat-completions server that answers with the given
// message contents in order; the last response repeats for extra calls.
func NewServer(t *testing.T, responses ...string) *Server {
	t.Helper()
	s := &Server{responses: responses}

	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.enter()
		defer s.leave()

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
		var content string
		switch {
		case s.EchoPrompt:
			content = lastUserMessage(req)
		case len(s.responses) > 0:
			content = s.responses[idx]
		}
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

func (s *Server) enter() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inFlight++
	if s.inFlight > s.maxInFlt {
		s.maxInFlt = s.inFlight
	}
}

func (s *Server) leave() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inFlight--
}

// MaxConcurrent returns the peak number of requests handled simultaneously.
func (s *Server) MaxConcurrent() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxInFlt
}

// lastUserMessage returns the content of the final user message in the request,
// used by EchoPrompt mode to make responses row-identifiable.
func lastUserMessage(req map[string]interface{}) string {
	messages, _ := req["messages"].([]interface{})
	for i := len(messages) - 1; i >= 0; i-- {
		msg, ok := messages[i].(map[string]interface{})
		if !ok {
			continue
		}
		if msg["role"] == "user" {
			content, _ := msg["content"].(string)
			return content
		}
	}
	return ""
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
