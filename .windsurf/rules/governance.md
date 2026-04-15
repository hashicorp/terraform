---
trigger: always_on
description: Governance rules for terraform — compiled from governance.md by crag
---

# Windsurf Rules — terraform

Generated from governance.md by crag. Regenerate: `crag compile --target windsurf`

## Project

(No description)

**Stack:** go, docker

## Runtimes

go

## Cascade Behavior

When Windsurf's Cascade agent operates on this project:

- **Always read governance.md first.** It is the single source of truth for quality gates and policies.
- **Run all mandatory gates before proposing changes.** Stop on first failure.
- **Respect classifications.** OPTIONAL gates warn but don't block. ADVISORY gates are informational.
- **Respect path scopes.** Gates with a `path:` annotation must run from that directory.
- **No destructive commands.** Never run rm -rf, dd, DROP TABLE, force-push to main, curl|bash, docker system prune.
- - No hardcoded secrets — grep for sk_live, AKIA, password= before commit
- Follow the project commit conventions.

## Quality Gates (run in order)

1. `go vet ./...`
2. `go test ./...`
3. `go test -race -timeout=30m -v ./tfexec/internal/e2etest`
4. `go test -cover "./...")`
5. `go test -race ./internal/terraform ./internal/command ./internal/states`
6. `make syncdeps`
7. `make fmtcheck importscheck vetcheck copyright generate staticcheck exhaustive protobuf`
8. `go build  # from .github/CONTRIBUTING.md`

## Rules of Engagement

1. **Minimal changes.** Don't rewrite files that weren't asked to change.
2. **No new dependencies** without explicit approval.
3. **Prefer editing** existing files over creating new ones.
4. **Always explain** non-obvious changes in commit messages.
5. **Ask before** destructive operations (delete, rename, migrate schema).

---

**Tool:** crag — https://www.npmjs.com/package/@whitehatd/crag
