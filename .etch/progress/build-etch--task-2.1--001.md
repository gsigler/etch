# Session: Task 2.1 – API client
**Plan:** build-etch
**Task:** 2.1
**Session:** 001
**Started:** 2026-02-15 15:40
**Status:** pending

## Changes Made
<!-- List files created or modified -->

## Acceptance Criteria Updates
- [ ] Makes Messages API requests successfully
- [ ] Streaming SSE output works (yields text chunks via channel or callback)
- [ ] Handles auth, rate limit, server, and network errors with clear messages
- [ ] Respects model setting from config
- [ ] Exponential backoff on 429 (1s, 2s, 4s, give up)
- [ ] Tests using `httptest.NewServer` — no real API calls: success response, streaming chunks, 401/429/500 error handling, backoff retry count
- [ ] `go test ./internal/api/...` passes

## Decisions & Notes
<!-- Design decisions, important context for future sessions -->

## Blockers
<!-- Anything blocking progress -->

## Next
<!-- What still needs to happen -->
