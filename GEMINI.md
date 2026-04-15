<!-- crag:auto-start -->
# GEMINI.md

> Generated from governance.md by crag. Regenerate: `crag compile --target gemini`

## Project Context

- **Name:** terraform
- **Stack:** go, docker
- **Runtimes:** go

## Rules

### Quality Gates

Run these checks in order before committing any changes:

1. [lint] `go vet ./...`
2. [test] `go test ./...`
3. [ci (inferred from workflow)] `go test -race -timeout=30m -v ./tfexec/internal/e2etest`
4. [ci (inferred from workflow)] `go test -cover "./...")`
5. [ci (inferred from workflow)] `go test -race ./internal/terraform ./internal/command ./internal/states`
6. [ci (inferred from workflow)] `make syncdeps`
7. [ci (inferred from workflow)] `make fmtcheck importscheck vetcheck copyright generate staticcheck exhaustive protobuf`
8. [contributor docs (advisory — confirm before enforcing)] `go build  # from .github/CONTRIBUTING.md`

### Security

- No hardcoded secrets — grep for sk_live, AKIA, password= before commit

### Workflow

- Follow project commit conventions
- Run quality gates before committing
- Review security implications of all changes

<!-- crag:auto-end -->
