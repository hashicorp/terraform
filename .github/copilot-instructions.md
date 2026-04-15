<!-- crag:auto-start -->
# Copilot Instructions — terraform

> Generated from governance.md by crag. Regenerate: `crag compile --target copilot`



**Stack:** go, docker

## Runtimes

go

## Quality Gates

When you propose changes, the following checks must pass before commit:

- **lint**: `go vet ./...`
- **test**: `go test ./...`
- **ci (inferred from workflow)**: `go test -race -timeout=30m -v ./tfexec/internal/e2etest`
- **ci (inferred from workflow)**: `go test -cover "./...")`
- **ci (inferred from workflow)**: `go test -race ./internal/terraform ./internal/command ./internal/states`
- **ci (inferred from workflow)**: `make syncdeps`
- **ci (inferred from workflow)**: `make fmtcheck importscheck vetcheck copyright generate staticcheck exhaustive protobuf`
- **contributor docs (advisory — confirm before enforcing)**: `go build  # from .github/CONTRIBUTING.md`

## Expectations for AI-Assisted Code

1. **Run gates before suggesting a commit.** If you cannot run them (no shell access), explicitly remind the human to run them.
2. **Respect classifications.** `MANDATORY` gates must pass. `OPTIONAL` gates should pass but may be overridden with a note. `ADVISORY` gates are informational only.
3. **Respect workspace paths.** When a gate is scoped to a subdirectory, run it from that directory.
4. **No hardcoded secrets.** - No hardcoded secrets — grep for sk_live, AKIA, password= before commit
5. Follow project commit conventions.
6. **Conservative changes.** Do not rewrite unrelated files. Do not add new dependencies without explaining why.

## Tool Context

This project uses **crag** (https://www.npmjs.com/package/@whitehatd/crag) as its AI-agent governance layer. The `governance.md` file is the authoritative source. If you have shell access, run `crag check` to verify the infrastructure and `crag diff` to detect drift.

<!-- crag:auto-end -->
