# Repository Guidelines

## Project Structure & Module Organization
- Source is organized by component: `function/`, `http/`, `kv/`, `log/`, `metrics/`, `sql/`.
- Each component has a `mock/` subpackage for unit tests and examples.
- Core entry points live at the repo root (e.g., `sdk.go`).
- Tests are colocated with code and named `*_test.go`.

## Build, Test, and Development Commands
- `make build`: Build all components.
- `make tests`: Run unit tests for all components with race detection and coverage.
- `make benchmarks`: Run Go benchmarks across components.
- `make format`: Apply `gofmt -s -w` to the repo and components.
- `make lint`: Run `golangci-lint` (if installed) for all modules.
- `make clean`: Remove coverage, test binaries, and vendor dirs.
- Per-module usage: `make -C http tests`, or `go test ./http/...`.

## Coding Style & Naming Conventions
- Language: Go (Go 1.21+ supported in CI). Use tabs and standard Go formatting.
- Formatting: `gofmt -s` (enforced by `make format`). Optional `golangci-lint` for lint rules.
- Packages and files: lowercase, short, and meaningful (e.g., `http/mock`).
- Exported identifiers: `CamelCase`. Tests follow `TestXxx(t *testing.T)`.

## Testing Guidelines
- Framework: standard library `testing` with `go test`.
- Coverage: component `Makefile`s produce `coverage.out` and `coverage.html`.
- Use component `mock/` packages to isolate host interactions.
- Naming: unit tests in `*_test.go`; benchmarks as `BenchmarkXxx(b *testing.B)`.
- Example: `make -C kv tests` (race + coverage) or `go test -race -cover ./kv/...`.

## Commit & Pull Request Guidelines
- Commit convention: Conventional Commits (see `.github/COMMIT_CONVENTION.md`). Example: `fix(http): correct header parsing`.
- Before opening a PR: run `make format`, `make lint`, and `make tests` locally.
- PRs should include: concise description, affected component(s), linked issues, and test updates/coverage for new behavior.
- CI: commit lint, multi-version tests, and lint run on PRs; coverage is uploaded automatically.

## Security & Configuration Tips
- Do not commit secrets or tokens. Use environment variables locally; CI handles Codecov token.
- Keep dependencies up to date; Dependabot is configured for all modules.
