# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

DS2API converts DeepSeek Web chat capabilities into OpenAI, Claude, and Gemini compatible APIs. The backend is Go (`cmd/ds2api/`, `api/`, `internal/`), with a small Node.js runtime for Vercel streaming (`api/chat-stream.js`). The admin WebUI is React (`webui/`), built to `static/admin` at deploy time.

Module name: `ds2api`. Go 1.26+ required. Node 20.19+/22.12+ only needed to build the WebUI.

## Commands

```bash
# Run the server (default port 5001)
go run ./cmd/ds2api

# Build
go build ./cmd/ds2api

# Run all unit tests
go test ./...

# Run a single package's tests
go test ./internal/<package> -count=1

# Run a single test case
go test -v -run TestName ./internal/<package>

# Run Node unit tests
./tests/scripts/run-unit-node.sh

# WebUI dev server
npm run dev --prefix webui

# WebUI production build
npm run build --prefix webui

# Format Go files
gofmt -w <changed-go-files>

# End-to-end tests (needs real accounts in config.json)
go run ./cmd/ds2api-tests
./tests/scripts/run-live.sh

# E2E tests skipping preflight checks
go run ./cmd/ds2api-tests --no-preflight

# Docker
docker-compose up -d
```

### PR gates (run before opening/updating a PR)

```bash
./scripts/lint.sh
./tests/scripts/check-refactor-line-gate.sh
./tests/scripts/run-unit-all.sh
npm run build --prefix webui
```

These mirror `.github/workflows/quality-gates.yml`. CI also runs Go unit tests on macOS/Windows and a release-target cross-build check on push to `dev`/`main`.

## Architecture

### Request flow

Every API request flows through the same pipeline:

1. **Router** (`internal/server/router.go`) — chi router with CORS, RealIP, RequestID, Logger, Recoverer middleware
2. **Protocol adapter** (`internal/httpapi/openai/`, `claude/`, `gemini/`, `ollama/`) — normalizes protocol-specific request shapes
3. **PromptCompat** (`internal/promptcompat/`) — the core compatibility layer. Converts structured API messages/tools/attachments into DeepSeek web-chat plain-text context (`prompt` string + `ref_file_ids` + control bits). This is the most important compatibility artifact in the project.
4. **Completion runtime** (`internal/completionruntime/`) — shared DeepSeek session/PoW/completion execution, empty-output retry, account-level fresh retry before 429
5. **Assistant turn** (`internal/assistantturn/`) — normalizes upstream SSE output into unified assistant turn semantics (thinking, tool calls, citations, usage, stop/error)
6. **Format layer** (`internal/format/`) — renders back to the target protocol (OpenAI, Claude)

### Key internal packages

| Package | Responsibility |
| --- | --- |
| `internal/deepseek/client` | Upstream DeepSeek calls: auth, session, completion, file upload/delete |
| `internal/deepseek/protocol` | DeepSeek URL constants, SSE constants (shared JSON with Node side) |
| `internal/deepseek/transport` | HTTP transport layer (proxy, TLS) |
| `internal/account` | Account pool, per-account concurrency slots, waiting queue |
| `internal/auth` | API key/bearer/JWT credential resolution, admin key validation |
| `internal/config` | Config loading, validation, hot-reload settings |
| `internal/toolcall` | DSML/XML tool call parsing and repair (canonical + DSML pipe-delimited forms) |
| `internal/toolstream` | Streaming tool call leak prevention and incremental detection |
| `internal/stream` + `internal/sse` | Unified stream consumption and SSE parsing |
| `internal/prompt` | Prompt assembly helpers |
| `internal/chathistory` | Server-side chat history persistence and query |
| `internal/responsehistory` | Raw upstream response archiving (before protocol translation) |
| `internal/translatorcliproxy` | Claude/Gemini ↔ OpenAI structural bridge — only for Vercel prepare/release, fallback, and regression tests |
| `internal/devcapture` | In-memory packet capture for debugging |
| `internal/compat` | Compatibility regression tests using SSE fixtures |
| `pow/` | Standalone PoW (DeepSeekHashV1) implementation and benchmarks |

### Vercel streaming path

`/v1/chat/completions` on Vercel is rewritten to Node (`api/chat-stream.js`). Auth, account lease, and completion payload are prepared by Go; the Node side does real-time SSE forwarding with Go-aligned tool sieve semantics. Internal packages `internal/js/chat-stream/` and `internal/js/helpers/stream-tool-sieve/` are the Go-side counterparts.

## Development rules (from AGENTS.md)

- **Gofmt**: Run `gofmt -w` on every changed Go file before commit.
- **Cleanup errors**: Do not ignore error returns from `Close`, `Flush`, `Sync`, or similar I/O cleanup calls. If you can't return the error, log it.
- **Scope**: Keep changes additive and tightly scoped. Don't mix unrelated refactors into feature PRs.
- **Protocol boundary**: Never let protocol adapters (OpenAI Chat, Responses, Claude, Gemini) own shared business behavior. Normalize protocol-specific requests first, run shared logic in one place, then render back. Business logic that must stay globally consistent includes: empty-output retry, thinking/reasoning handling, tool-call detection and policy, usage accounting, current-input-file injection, history persistence, file/reference handling, and completion payload assembly.
- **Documentation sync**: When business logic or user-visible behavior changes, update the corresponding docs in the same change. `docs/prompt-compatibility.md` is the source-of-truth for the API→web-chat text compatibility flow — update it for any change affecting message normalization, tool prompt injection, tool history, file/reference handling, history split, or completion payload assembly.

## Tool call semantics

Executable tool calls use half-width pipe-delimited DSML form: `<|DSML|tool_calls><|DSML|invoke name="..."><|DSML|parameter name="...">`. Legacy canonical XML (`<tool_calls><invoke name="..."><parameter name="...">`) is also accepted. Other forms (`<tools>`, `<tool_call>`, `<function_call>`, `tool_use`, plain JSON fragments) are treated as plain text. See `docs/toolcall-semantics.md` for details.

## Configuration

`config.example.json` is the template. At runtime, use either a `config.json` file or the `DS2API_CONFIG_JSON` env var (Base64-encoded JSON). `keys`/`api_keys` are client access credentials; `accounts` are DeepSeek upstream accounts (email or mobile login). Model aliases (OpenAI/Claude/Gemini) are configured in `model_aliases`.

Key env vars: `DS2API_ADMIN_KEY`, `DS2API_CONFIG_PATH`, `DS2API_CONFIG_JSON`, `PORT` (default 5001), `DS2API_DEV_PACKET_CAPTURE`.
