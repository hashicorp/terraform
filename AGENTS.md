<!-- crag:auto-start -->
# AGENTS.md

> Generated from governance.md by crag. Regenerate: `crag compile --target agents-md`

## Project: terraform


## Quality Gates

All changes must pass these checks before commit:

### Lint
1. `go vet ./...`

### Test
1. `go test ./...`

### Ci (inferred from workflow)
1. `go test -race -timeout=30m -v ./tfexec/internal/e2etest`
2. `go test -cover "./...")`
3. `go test -race ./internal/terraform ./internal/command ./internal/states`
4. `make syncdeps`
5. `make fmtcheck importscheck vetcheck copyright generate staticcheck exhaustive protobuf`

### Contributor docs (advisory — confirm before enforcing)
1. `go build  # from .github/CONTRIBUTING.md`

## Coding Standards

- Stack: go, docker
- Follow project commit conventions

## Architecture

- Type: monolith

## Key Directories

- `.github/` — CI/CD
- `docs/` — documentation
- `scripts/` — tooling
- `tools/` — tooling

## Testing

- Framework: go test
- Layout: flat

## Anti-Patterns

Do not:
- Do not ignore returned errors — handle or explicitly discard with `_ =`
- Do not use `panic()` in library code — return errors instead
- Do not use `init()` functions unless absolutely necessary
- Do not use `latest` tag in FROM — pin to a specific version
- Do not run containers as root — use a non-root USER

## Security

- No hardcoded secrets — grep for sk_live, AKIA, password= before commit

## Workflow

1. Read `governance.md` at the start of every session — it is the single source of truth.
2. Run all mandatory quality gates before committing.
3. If a gate fails, fix the issue and re-run only the failed gate.
4. Use the project commit conventions for all changes.

<!-- crag:auto-end -->
