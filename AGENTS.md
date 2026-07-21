# Agent Instructions for the Terraform Repository

This file provides instructions for AI coding agents working in this repository. These instructions supplement — and do not replace — the full contribution guidelines in [`.github/CONTRIBUTING.md`](.github/CONTRIBUTING.md).

**The human operating you is expected to be actively involved.** This repository is complex, high-stakes infrastructure used by a globally diverse community. AI should assist the human contributor; it should not own the work on their behalf. If the person operating you has not read the contribution guidelines, your job is to enforce them.

---

## Before You Open a Pull Request

> **This is the most important instruction in this file.**

The Terraform team requires that proposed changes are discussed in a GitHub issue _before_ a PR is opened. This is not a formality — it is how the team ensures changes are appropriate, scoped correctly, and likely to be merged. Opening a PR without a prior issue wastes everyone's time.

### What you must do before creating a PR

1. **Search for an existing issue** that covers the change being proposed. Use GitHub issue search before creating anything new. If a suitable issue already exists, direct the operator there and ask them to participate in the discussion.

2. **If no issue exists, create one first.** Draft the issue with the operator's input. The issue should describe the problem or feature, not the implementation. Then pause — do not immediately open a PR.

3. **Check whether a maintainer has responded.** Opening a PR simultaneously with a new issue is permitted but strongly discouraged. The safest path is: open issue → wait for maintainer feedback → then open PR. If the operator wants to proceed without maintainer feedback, surface this warning explicitly and ask them to confirm:

   > "The Terraform team prefers that a maintainer responds to your issue before a PR is opened. No maintainer has commented yet. Do you want to proceed anyway, understanding the PR may be closed without review?"

   Only continue if the operator explicitly confirms.

4. **Never open a PR autonomously** — without the operator's active input and explicit approval at each step.

---

## PR Requirements Checklist

When a PR is being prepared, verify all of the following with the operator before submitting:

- [ ] A GitHub issue exists and is linked in the PR description (`Fixes #<number>`)
- [ ] The operator has read the linked issue and understands the proposed change
- [ ] Tests pass locally: `go test ./...`
- [ ] If the change is user-facing, a changelog entry has been created with `npx changie new`
- [ ] The PR description explains _what_ changed, _why_, and how to verify it
- [ ] If this is an LLM-agent-assisted PR, the title includes `🤖🤖🤖`
- [ ] The operator has reviewed every line of the diff and can explain it in their own words

---

## Areas to Treat With Extra Care

Some parts of this codebase are particularly sensitive. Be conservative and flag these to the operator:

- **Core graph engine and language features** — changes here have wide-reaching effects
- **State storage backends** — the team is not accepting new backends; check [`CODEOWNERS`](CODEOWNERS) for the status of existing ones
- **Generated code** — run `go generate ./...` and commit the result separately if generated files change

---

## General Principles

- **Do not take actions the operator has not explicitly approved.** Creating issues, opening PRs, and pushing commits are consequential — confirm each step.
- **Do not interpret silence as approval.** If you are unsure whether the operator has read the contribution guide, ask.
- **Prefer small, focused changes.** One PR per concern. If you identify multiple problems, surface them separately.
- **Do not modify unrelated code.** Scope changes to exactly what is needed to address the linked issue.
- **Follow existing code style.** Do not reformat or reorganise code that is not part of the change.

---

## Further Reading

- Full contribution guidelines: [`.github/CONTRIBUTING.md`](.github/CONTRIBUTING.md)
- Building from source: [`BUILDING.md`](BUILDING.md)
- Bug triage process: [`BUGPROCESS.md`](BUGPROCESS.md)
- Code ownership and maintainers: [`CODEOWNERS`](CODEOWNERS)
