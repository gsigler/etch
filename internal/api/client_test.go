package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSend_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key=test-key, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("expected anthropic-version=2023-06-01, got %s", r.Header.Get("anthropic-version"))
		}
		if r.Header.Get("content-type") != "application/json" {
			t.Errorf("expected content-type=application/json, got %s", r.Header.Get("content-type"))
		}

		var req messagesRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Model != "test-model" {
			t.Errorf("expected model=test-model, got %s", req.Model)
		}
		if req.System != "you are helpful" {
			t.Errorf("expected system prompt, got %s", req.System)
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "hello" {
			t.Errorf("unexpected messages: %+v", req.Messages)
		}

		json.NewEncoder(w).Encode(MessagesResponse{
			Content: []contentBlock{{Type: "text", Text: "world"}},
		})
	}))
	defer srv.Close()

	c := NewClient("test-key", "test-model")
	c.BaseURL = srv.URL

	text, err := c.Send("you are helpful", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "world" {
		t.Errorf("expected 'world', got %q", text)
	}
}

func TestSend_RespectsModel(t *testing.T) {
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req messagesRequest
		json.NewDecoder(r.Body).Decode(&req)
		gotModel = req.Model
		json.NewEncoder(w).Encode(MessagesResponse{
			Content: []contentBlock{{Type: "text", Text: "ok"}},
		})
	}))
	defer srv.Close()

	c := NewClient("key", "claude-opus-4-20250514")
	c.BaseURL = srv.URL
	c.Send("", "test")

	if gotModel != "claude-opus-4-20250514" {
		t.Errorf("expected model claude-opus-4-20250514, got %s", gotModel)
	}
}

func TestSendStream_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req messagesRequest
		json.NewDecoder(r.Body).Decode(&req)
		if !req.Stream {
			t.Error("expected stream=true")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		chunks := []string{"Hello", " ", "world"}
		for _, chunk := range chunks {
			evt := map[string]interface{}{
				"type": "content_block_delta",
				"delta": map[string]string{
					"type": "text_delta",
					"text": chunk,
				},
			}
			data, _ := json.Marshal(evt)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	c := NewClient("key", "model")
	c.BaseURL = srv.URL

	var chunks []string
	text, err := c.SendStream("sys", "hi", func(chunk string) {
		chunks = append(chunks, chunk)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", text)
	}
	if len(chunks) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(chunks))
	}
}

func TestSendStream_SkipsNonTextEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// message_start event (should be skipped)
		fmt.Fprintf(w, "data: {\"type\":\"message_start\"}\n\n")
		// content_block_start (should be skipped)
		fmt.Fprintf(w, "data: {\"type\":\"content_block_start\"}\n\n")
		// actual text delta
		fmt.Fprintf(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"hi\"}}\n\n")
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	c := NewClient("key", "model")
	c.BaseURL = srv.URL

	text, err := c.SendStream("", "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "hi" {
		t.Errorf("expected 'hi', got %q", text)
	}
}

func TestSend_AuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer srv.Close()

	c := NewClient("bad-key", "model")
	c.BaseURL = srv.URL

	_, err := c.Send("", "hello")
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "invalid API key") {
		t.Errorf("expected helpful error message, got %q", err.Error())
	}
}

func TestSend_RateLimitRetry(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(429)
			w.Write([]byte("rate limited"))
			return
		}
		json.NewEncoder(w).Encode(MessagesResponse{
			Content: []contentBlock{{Type: "text", Text: "ok"}},
		})
	}))
	defer srv.Close()

	c := NewClient("key", "model")
	c.BaseURL = srv.URL
	c.InitialBackoff = 10 * time.Millisecond

	start := time.Now()
	text, err := c.Send("", "hello")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if text != "ok" {
		t.Errorf("expected 'ok', got %q", text)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	// Should have waited at least 10ms+20ms = 30ms for backoff
	if elapsed < 30*time.Millisecond {
		t.Errorf("expected at least 30ms of backoff, elapsed %v", elapsed)
	}
}

func TestSend_RateLimitExhausted(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(429)
		w.Write([]byte("rate limited"))
	}))
	defer srv.Close()

	c := NewClient("key", "model")
	c.BaseURL = srv.URL
	c.InitialBackoff = 1 * time.Millisecond

	_, err := c.Send("", "hello")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("expected rate limit error, got %q", err.Error())
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestSend_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	c := NewClient("key", "model")
	c.BaseURL = srv.URL

	_, err := c.Send("", "hello")
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status 500, got: %v", err)
	}
}

func TestSend_NetworkError(t *testing.T) {
	c := NewClient("key", "model")
	c.BaseURL = "http://127.0.0.1:1" // nothing listening
	c.HTTPClient.Timeout = 1 * time.Second

	_, err := c.Send("", "hello")
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "sending request") {
		t.Errorf("expected 'sending request' in error, got %q", err.Error())
	}
}
