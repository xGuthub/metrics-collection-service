# Repository Guidelines

## Project Structure & Module Organization
- The application must follow the principles of clean architecture and SOLID.
- `cmd/server` – HTTP API service entrypoint (`main.go`, middleware).
- `cmd/agent` – metrics agent entrypoint (`main.go`).
- `internal/{config,handler,logger,model,repository,service}` – application code (flags/env, HTTP handlers, logging, domain models, storage, business logic).
- `api`, `pkg`, `migrations` – auxiliary docs/packages and DB artifacts.
- Tests live next to code as `*_test.go` (e.g., `internal/handler/...`).

## Build, Test, and Development Commands
- Build server: `go build -o cmd/server/server ./cmd/server`
- Build agent: `go build -o cmd/agent/agent ./cmd/agent`
- Run server (examples): `go run ./cmd/server -a=localhost:8080 -i=300 -f=/tmp/metrics-db.json -r=true`
- Run agent (examples): `go run ./cmd/agent -a=localhost:8080 -r=10 -p=2`
- Unit tests: `go test ./...`
- Static checks: `go vet -vettool=statictest ./...`
- Formatting: `go fmt ./...` and `goimports -w .`

## Coding Style & Naming Conventions
- Go 1.24.x. Use `gofmt`/`goimports` before committing.
- Indentation: tabs; line length: reasonable; avoid unnecessary blank lines.
- Packages: lower_snake or single word; exported identifiers use `CamelCase`, unexported `camelCase`.
- Filenames: lower_snake, one type per file where practical.

## Testing Guidelines
- Framework: standard `testing` with table-driven tests when suitable.
- Name tests `TestXxx` in `*_test.go`. Example: `go test ./internal/handler -run TestHomeHandler_Empty`.
- Keep handler/service logic covered; prefer tests near the code they verify.

## Commit & Pull Request Guidelines
- Branch naming: `iter<number>` (e.g., `iter7`) or `master` to satisfy CI rules.
- Commit style: short imperative summary, optional body (e.g., “add gzip support”). Group related changes.
- PRs must include: purpose, scope, any config/env changes, and link to issues. Add screenshots or curl examples for API changes.
- CI must pass (`go vet` with `statictest`, autotests building `cmd/server` and `cmd/agent`).

## Security & Configuration Tips
- Config via flags and env: `ADDRESS`, `STORE_INTERVAL`, `FILE_STORAGE_PATH`, `RESTORE`, `REPORT_INTERVAL`, `POLL_INTERVAL`.
- Do not commit binaries; `.gitignore` excludes `*.exe` and built outputs.
- For local persistence, ensure `-f` path is writable; set `-i=0` to disable autosave.
