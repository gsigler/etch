# Session: Task 2.1 – API client
**Plan:** build-etch
**Task:** 2.1
**Session:** 002
**Started:** 2026-02-15 18:11
**Status:** completed

## Changes Made
- Created `internal/api/client.go` — Anthropic Messages API client
- Created `internal/api/client_test.go` — 9 tests, all passing

## Acceptance Criteria Updates
- [x] Makes Messages API requests successfully
- [x] Streaming SSE output works (yields text chunks via channel or callback)
- [x] Handles auth, rate limit, server, and network errors with clear messages
- [x] Respects model setting from config
- [x] Exponential backoff on 429 (1s, 2s, 4s, give up)
- [x] Tests using `httptest.NewServer` — no real API calls: success response, streaming chunks, 401/429/500 error handling, backoff retry count
- [x] `go test ./internal/api/...` passes

## Decisions & Notes
- `Client` struct has public fields: APIKey, Model, BaseURL, MaxTokens, HTTPClient, InitialBackoff
- `BaseURL` and `InitialBackoff` are overridable for testing (BaseURL points to httptest server, InitialBackoff speeds up retry tests)
- `Send()` for non-streaming, `SendStream()` for streaming with a `StreamCallback func(text string)` — both return full accumulated text
- SSE parsing handles content_block_delta events with text_delta type, skips all other event types
- `APIError` type with StatusCode and Message for structured error handling
- 401 returns immediately with helpful message about checking API key
- 429 retries with exponential backoff (configurable initial, doubles each time, max 3 attempts)
- 500+ returns immediately with response body as error message
- Network errors wrapped with "sending request:" prefix
- Default max_tokens: 8192, timeout: 5 minutes

## Blockers
None.

## Next
Task complete. Ready for Task 2.2 (Plan generation command) which depends on this.
