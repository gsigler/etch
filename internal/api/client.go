package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	etcherr "github.com/gsigler/etch/internal/errors"
)

const (
	defaultBaseURL      = "https://api.anthropic.com"
	messagesPath        = "/v1/messages"
	anthropicVersion    = "2023-06-01"
	defaultMaxTokens    = 8192
	maxRetries          = 3
	initialRetryBackoff = 1 * time.Second
)

// Client is an HTTP client for the Anthropic Messages API.
type Client struct {
	APIKey          string
	Model           string
	BaseURL         string
	MaxTokens       int
	HTTPClient      *http.Client
	InitialBackoff  time.Duration // for testing; defaults to 1s
}

// NewClient creates a Client with sensible defaults.
func NewClient(apiKey, model string) *Client {
	return &Client{
		APIKey:     apiKey,
		Model:      model,
		BaseURL:    defaultBaseURL,
		MaxTokens:  defaultMaxTokens,
		HTTPClient: &http.Client{Timeout: 5 * time.Minute},
	}
}

// messagesRequest is the request body for the Messages API.
type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []message `json:"messages"`
	Stream    bool      `json:"stream,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MessagesResponse is the response from a non-streaming Messages API call.
type MessagesResponse struct {
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Text returns the concatenated text from all text content blocks.
func (r MessagesResponse) Text() string {
	var parts []string
	for _, b := range r.Content {
		if b.Type == "text" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "")
}

// APIError represents an error response from the Anthropic API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("anthropic api error (status %d): %s", e.StatusCode, e.Message)
}

// Send makes a non-streaming Messages API request and returns the response text.
func (c *Client) Send(system, userMessage string) (string, error) {
	body := messagesRequest{
		Model:     c.Model,
		MaxTokens: c.maxTokens(),
		System:    system,
		Messages:  []message{{Role: "user", Content: userMessage}},
	}

	resp, err := c.doWithRetry(body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result MessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", etcherr.WrapAPI("decoding API response", err)
	}
	return result.Text(), nil
}

// StreamCallback is called with each text chunk during streaming.
type StreamCallback func(text string)

// SendStream makes a streaming Messages API request and calls cb with each text delta.
// Returns the full accumulated text.
func (c *Client) SendStream(system, userMessage string, cb StreamCallback) (string, error) {
	body := messagesRequest{
		Model:     c.Model,
		MaxTokens: c.maxTokens(),
		System:    system,
		Messages:  []message{{Role: "user", Content: userMessage}},
		Stream:    true,
	}

	resp, err := c.doWithRetry(body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return parseSSE(resp.Body, cb)
}

// parseSSE reads SSE data lines from r, extracts text deltas, and calls cb for each.
func parseSSE(r io.Reader, cb StreamCallback) (string, error) {
	var full strings.Builder
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue // skip malformed events
		}
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			full.WriteString(event.Delta.Text)
			if cb != nil {
				cb(event.Delta.Text)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return full.String(), etcherr.WrapAPI("reading stream", err)
	}
	return full.String(), nil
}

func (c *Client) doWithRetry(body messagesRequest) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, etcherr.WrapAPI("encoding request", err)
	}

	backoff := c.InitialBackoff
	if backoff == 0 {
		backoff = initialRetryBackoff
	}
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest("POST", c.BaseURL+messagesPath, bytes.NewReader(payload))
		if err != nil {
			return nil, etcherr.WrapAPI("creating HTTP request", err)
		}
		req.Header.Set("x-api-key", c.APIKey)
		req.Header.Set("anthropic-version", anthropicVersion)
		req.Header.Set("content-type", "application/json")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, etcherr.WrapAPI("sending request", err).
				WithHint("check your network connection")
		}

		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			return resp, nil
		case resp.StatusCode == 401:
			drainBody(resp)
			return nil, etcherr.API("invalid API key").
				WithHint("check your ANTHROPIC_API_KEY env var or api_key in .etch/config.toml")
		case resp.StatusCode == 429:
			drainBody(resp)
			if attempt < maxRetries-1 {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			return nil, etcherr.API("rate limited — retries exhausted").
				WithHint("wait a moment and try again")
		default:
			msg := drainBodyString(resp)
			return nil, etcherr.API(fmt.Sprintf("API error (status %d): %s", resp.StatusCode, msg))
		}
	}
	// unreachable, but satisfy the compiler
	return nil, etcherr.API("request failed after retries")
}

func (c *Client) maxTokens() int {
	if c.MaxTokens > 0 {
		return c.MaxTokens
	}
	return defaultMaxTokens
}

func drainBody(resp *http.Response) {
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func drainBodyString(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if len(b) > 0 {
		return string(b)
	}
	return http.StatusText(resp.StatusCode)
}
