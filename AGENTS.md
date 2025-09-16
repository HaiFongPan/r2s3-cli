# Repository Guidelines

## Project Structure & Module Organization
- Root entrypoint: `main.go`; CLI commands in `cmd/` (`upload.go`, `list.go`, `delete.go`).
- Core packages in `internal/`:
  - `internal/config/` (Viper-based config, env `R2CLI_*`).
  - `internal/r2/` (R2/S3 client).
  - `internal/tui/` (Bubble Tea UI, themes/styles).
  - `internal/utils/` (IO, progress, MIME helpers).
- Examples and docs in `examples/`, `docs/`; images in `images/`.

## Build, Test, and Development Commands
- Build: `go build -o r2s3-cli` (Go 1.25+).
- Run TUI: `go run .` or `./r2s3-cli`.
- Run command: `go run . upload file.jpg`.
- Tests: `go test ./...` (use `-v` or `-cover` as needed).
- Lint/basic checks: `go fmt ./...` and `go vet ./...`.

## Coding Style & Naming Conventions
- Use `go fmt` before committing; no manual formatting tweaks.
- Package names: short, lowercase; files use underscores if needed (e.g., `file_uploader.go`).
- Exported identifiers use PascalCase and have doc comments; unexported use lowerCamelCase.
- Prefer small, focused functions; avoid unnecessary globals. Keep CLI flag parsing in `cmd/`, logic in `internal/`.

## Testing Guidelines
- Framework: Go `testing` with `testify` available. Place tests as `*_test.go` next to source.
- Favor table-driven tests. Name tests `TestPackage_Function_Scenario`.
- Coverage target: changed packages ≥ 80% statements; critical paths (`internal/config`, `internal/r2`, `internal/utils`) ≥ 90% where practical.
- Measure: `go test -coverprofile=cover.out ./... && go tool cover -func=cover.out`.

## Commit & Pull Request Guidelines
- Use Conventional Commits: `feat:`, `fix:`, `refactor(tui):`, `chore:`, `test:`.
  - Example: `fix(utils): handle 404 from HeadObject`
- PRs must include: concise description, rationale, test plan (commands run), and screenshots/GIFs for TUI changes (`images/` helpful).
- Link related issues. Keep diffs minimal and scoped to one concern.

## Security & Configuration Tips
- Never commit secrets. Use env vars (e.g., `export R2CLI_ACCESS_KEY_ID=...`).
- Default config path: `~/.r2s3-cli/config.toml` (see `examples/config.toml`).
- Validate with `R2CLI_*` envs before running: `R2CLI_BUCKET_NAME`, `R2CLI_ACCOUNT_ID`, etc.

## Architecture Overview
```
┌─────────┐      flags/cobra       ┌───────────────┐
│  cmd/   │ ─────────────────────▶ │ internal/*    │
│ (CLI)   │                        │  ├ config     │ Viper + env (R2CLI_*)
└────┬────┘                        │  ├ r2         │ AWS S3 SDK client
     │ go run/build                │  ├ tui        │ Bubble Tea models/views
     ▼                             │  └ utils      │ IO, progress, MIME
  main.go                          └──────┬────────┘
                                          │
                                          ▼
                                   Cloudflare R2 (S3 API)
```
Key flows: `cmd/` parses flags → delegates to `internal/*`; config merges defaults, file, and env; logging configured in `cmd/root.go`.

## Agent-Specific Instructions
- Make surgical changes; do not refactor unrelated code.
- Update or add nearest tests when changing behavior.
- Avoid adding new dependencies without discussion; prefer standard lib.
- Do not modify `LICENSE`. Keep logging non-intrusive (see `setupLogging`).
