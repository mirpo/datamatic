package llmtest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func post(t *testing.T, url, body string) map[string]interface{} {
	t.Helper()
	resp, err := http.Post(url+"/chat/completions", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	var decoded map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&decoded))
	return decoded
}

func TestServer_ScriptedResponsesAndCapture(t *testing.T) {
	srv := NewServer(t, "first", "second")

	resp1 := post(t, srv.URL, `{"model":"m1","temperature":0.5}`)
	resp2 := post(t, srv.URL, `{"model":"m1"}`)
	resp3 := post(t, srv.URL, `{"model":"m1"}`) // last response repeats

	content := func(r map[string]interface{}) string {
		choices := r["choices"].([]interface{})
		msg := choices[0].(map[string]interface{})["message"].(map[string]interface{})
		return msg["content"].(string)
	}

	assert.Equal(t, "first", content(resp1))
	assert.Equal(t, "second", content(resp2))
	assert.Equal(t, "second", content(resp3))

	assert.Equal(t, 3, srv.CallCount())
	require.Len(t, srv.Requests(), 3)
	assert.Equal(t, 0.5, srv.Requests()[0]["temperature"])
	assert.Equal(t, "m1", resp1["model"], "echoes request model to avoid mismatch warnings")
}
