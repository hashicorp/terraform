<!-- crag:auto-start -->
# CLAUDE.md — terraform

> Generated from governance.md by crag. Regenerate: `crag compile --target claude`



**Stack:** go, docker
**Runtimes:** go

## Quality Gates

Run these in order before committing. Stop on first MANDATORY failure:

- `go vet ./...`
- `go test ./...`
- `go test -race -timeout=30m -v ./tfexec/internal/e2etest`
- `go test -cover "./...")`
- `go test -race ./internal/terraform ./internal/command ./internal/states`
- `make syncdeps`
- `make fmtcheck importscheck vetcheck copyright generate staticcheck exhaustive protobuf`
- `go build  # from .github/CONTRIBUTING.md`

## Rules

1. Read `governance.md` at the start of every session — it is the single source of truth.
2. Run all mandatory quality gates before committing.
3. If a gate fails, attempt an automatic fix (lint/format) with bounded retry (max 2 attempts). If it still fails, escalate to the user.
4. Never modify files outside this repository.
5. Never run destructive system commands (`rm -rf /`, `DROP TABLE`, force-push to main).
- Follow project commit conventions

## Security

- No hardcoded secrets — grep for sk_live, AKIA, password= before commit

## Tool Context

This project uses **crag** (https://www.npmjs.com/package/@whitehatd/crag) as its governance engine. The `governance.md` file is the authoritative source. Run `crag audit` to detect drift and `crag compile --target all` to recompile all targets.

<!-- crag:auto-end -->
