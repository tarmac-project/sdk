# Repository Guidelines

## Project Structure & Module Organization
- Root package `sdk` exposes runtime config and handler registration; tests live alongside (`sdk_test.go`).
- `httpclient/` holds the HTTP capability client, benchmarks, and host-aware tests. Coverage artifacts may appear here but should be ignored.
- `hostmock/` simulates waPC host calls for integration-style assertions.
- CI, release, and automation configs reside under `.github/` and `.release-*`.

## Build, Test, and Development Commands
- `make build` — compiles each component (`httpclient`, etc.) via their local Makefiles.
- `make tests` — runs `go test -race -covermode=atomic` for all components and emits coverage reports.
- `make benchmarks` — executes package benchmarks; use when tuning hot paths.
- `make format` / `make lint` — apply gofmt/goimports/golines and run `golangci-lint`.
- Per-package work: `make -C httpclient tests`, `make -C httpclient lint`, etc.

## Coding Style & Naming Conventions
- Go 1.24+: rely on `make format` to run `gofmt`, `goimports`, and `golines`; do not hand-format.
- Use descriptive package-level names; keep short identifiers scoped to tight loops or helpers.
- Follow sentinel error pattern (`ErrThing`) and wrap with `%w` when returning.
- File order: package, import, const, var, type, func (only include sections when needed).
- Optimize struct field order for alignment; prefer smaller packing and avoid padding.

## Testing Guidelines
- Standard Go `testing` package with table-driven cases; include happy and error paths.
- Prefer black-box assertions via `hostmock` or purpose-built fakes.
- Benchmarks live in `*_benchmark_test.go`; keep them deterministic.
- Ensure race detector and coverage succeed before opening a PR.

## Commit & Pull Request Guidelines
- Commit messages follow Conventional Commits (`type(scope): subject`), e.g., `fix(httpclient): guard nil handler`.
- Summaries should be imperative, lowercase start, and omit trailing periods.
- PRs should describe behavior changes, reference related issues, and note testing (`make tests`, `make lint`).

## Security & Configuration Tips
- No network or secret-aware tests should run by default; rely on mocks.
- Manage protobuf or waPC dependency bumps through Dependabot or curated PRs.
