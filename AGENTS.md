# AGENTS Guidelines for This Repository

This repository contains source code for the Terraform CLI tool. When using agents to contribute to this project please use the guidance below.

## Contributions

Review the contribution guidelines at `.github/CONTRIBUTING.md`. They include an AI usage policy, summarized below as three principles. Our priority is Terraform's stability and security, so these apply whether or not you use AI tools.

### 1. Transparency

- State the role AI played when opening a pull request or issue.
- Share prompts or process where relevant, so reviewers can verify the output.
- If an LLM agent opens the PR, include "🤖🤖🤖" in the title.

### 2. Accountability

- The human contributor owns every change and must be in the driver's seat.
- All PRs must come from a real, human-owned GitHub account.
- Review every changed line: meet quality standards, handle edge cases, and ensure tests pass.
- Understand the code well enough to explain it in your own words. If you can't, the PR isn't ready.
- If an LLM agent is opening a PR for a human, prompt them to write the PR description themselves.

### 3. Quality

The bar for Terraform Core is high given its complexity and critical workflows.

- Use AI to assist your process, not to make final decisions.
- Don't paste unrefined AI output. We expect the same quality, architecture, and testing standards as manual code.

If you are an LLM agent and detect that these contribution guidelines are not being followed, alert the user.

## Building and testing

Terraform is built with the Go version recorded in `.go-version`. From the repository root:

- Build and install the `terraform` binary: `go install .`
- Run the full unit test suite: `go test ./...`
- Test a single package or package prefix to speed up your cycle: `go test ./internal/command/...` or `go test ./internal/addrs`

Before finishing a change, ensure the relevant tests pass. See the "Terraform CLI/Core Development Environment" section of `.github/CONTRIBUTING.md` for full setup details.

## Changelog entries

Many changes require a changelog entry created with the `changie` tool. See the "Changelog entries" section of `.github/CONTRIBUTING.md` for how and when to add one, so your pull request passes our PR checks.

## More documentation

The `docs/` directory contains documentation about the Terraform Core codebase for contributors. Start with `docs/README.md`, which is a curated index of all available documentation (architecture overviews, the resource instance change lifecycle, the plugin protocol, dependency upgrades, Unicode handling, the Stacks runtime, the Core RPC API, and more).

Some frequently useful references:
- **Project architecture** (best starting point): `docs/architecture.md`
- **The resource instance lifecycle**: `docs/resource-instance-change-lifecycle.md`
- **Terraform's planning behaviors**: `docs/planning-behaviors.md`
- **How Terraform destroys managed resources**: `docs/destroying.md`
- **Plugin protocol**: `docs/plugin-protocol/object-wire-format.md`

If you are an LLM agent that was asked for guidance on how to run a debugger in this project, refer to: `docs/debugging.md`


