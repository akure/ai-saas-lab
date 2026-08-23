# Agent Rules for AI SaaS Lab

## 1. Context & Architecture
- **Tech Stack**: Go 1.25+, standard library centered, minimal third-party dependencies (Charm TUI, pgx, redis).
- **Directory Layout**:
  - `cmd/`: Application entrypoints (`cmd/lab`, `cmd/tui`). Keep `main` thin.
  - `internal/kernel/`: Core app wiring, state, config, and event bus.
  - `internal/modules/`: Feature modules (`auth`, `completion`, `billing`). Keep decoupled via kernel event bus.

## 2. Go Coding Standards
- **Idiomatic Go**:
  - Accept interfaces, return concrete types.
  - Explicit error handling (`if err != nil`). Always wrap errors with context (`fmt.Errorf("op: %w", err)`).
  - Pass `ctx context.Context` as first parameter for IO, network, or async functions.
  - Concurrency: protect shared state, handle goroutine lifecycles safely.
- **Dependencies**: Prefer stdlib. Do not add external packages without approval.

## 3. Execution & Verification
- **Inspect First**: Always check existing codebase/interfaces before adding or modifying code.
- **Verify**: Run `go test ./...` and verify build `go build ./cmd/...` after changes.
- **Token-Efficient Communication**: Keep explanations brief, code diffs minimal, and avoid fluff.
- **Media & Artifact Controls**: Do NOT commit video files (`.webm`, `.mp4`) or excessive PNG screenshots to Git repository. Playwright and browser automation test scripts MUST save test video recordings outside the project directory (e.g. in `os.tmpdir()` / `%TEMP%/ai-saas-lab-recordings/`) or keep them strictly ignored in `.gitignore`.
