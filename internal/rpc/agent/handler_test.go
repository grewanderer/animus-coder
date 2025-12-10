package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerStreamsEvents(t *testing.T) {
	handler := NewHandler(EchoRunner{}, nil)
	body := bytes.NewBufferString(`{"session_id":"test","prompt":"hello world"}`)
	req := httptest.NewRequest(http.MethodPost, "/agent/run", body)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	resp := rr.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	var eventCount int
	for scanner.Scan() {
		eventCount++
		var evt map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			t.Fatalf("invalid json event: %v", err)
		}
	}

	if eventCount == 0 {
		t.Fatalf("expected events, got none")
	}
}
