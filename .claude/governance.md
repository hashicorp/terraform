# Governance — terraform
# Inferred by crag analyze — review and adjust as needed

## Identity
- Project: terraform
- Stack: go, docker

## Gates (run in order, stop on failure)
### Lint
- go vet ./...

### Test
- go test ./...

### CI (inferred from workflow)
- go test -race -timeout=30m -v ./tfexec/internal/e2etest
- go test -cover "./...")
- go test -race ./internal/terraform ./internal/command ./internal/states
- make syncdeps
- make fmtcheck importscheck vetcheck copyright generate staticcheck exhaustive protobuf

### Contributor docs (ADVISORY — confirm before enforcing)
- go build  # from .github/CONTRIBUTING.md

## Advisories (informational, not enforced)
- hadolint Dockerfile  # [ADVISORY]
- actionlint  # [ADVISORY]

## Branch Strategy
- Trunk-based development
- Free-form commits
- Commit trailer: Co-Authored-By: Claude <noreply@anthropic.com>

## Security
- No hardcoded secrets — grep for sk_live, AKIA, password= before commit

## Autonomy
- Auto-commit after gates pass

## Deployment
- Target: docker
- CI: github-actions

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

## Dependencies
- Package manager: go (go.sum)
- Go: >=1.25.8

## Anti-Patterns

Do not:
- Do not ignore returned errors — handle or explicitly discard with `_ =`
- Do not use `panic()` in library code — return errors instead
- Do not use `init()` functions unless absolutely necessary
- Do not use `latest` tag in FROM — pin to a specific version
- Do not run containers as root — use a non-root USER

