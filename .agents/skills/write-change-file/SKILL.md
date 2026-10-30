---
name: write-change-file
description: Use when the user wants to create, write, or add a change file (changelog entry) for a Terraform PR or commit — guides producing a correctly formatted .changes/ YAML file.
---

# Write a Change File

Follow the style guide at `.changes/STYLE-GUIDE.md` for all content rules, field definitions, and
examples. This skill covers the interactive steps to gather information and produce the file.

## Step 0 — Is a change file even needed?

A changelog entry is only needed if a change is user-facing.

- Ask the user a yes/no question to confirm whether the change is user-facing.
- If the answer is no, stop creating a change file immediately and notify the user that one is not needed.

## Step 1 — Gather required information

Collect anything not already provided in the user's request:

1. **What changed?** Ask for a plain-language description of the change from a user's perspective.
1. **What kind of change is it?** One of: `NEW FEATURES`, `ENHANCEMENTS`, `BUG FIXES`, `NOTES`, `UPGRADE NOTES`, `BREAKING CHANGES`.
1. **What is the PR number?** (numeric only, e.g. `38397`). Note: the field is named `Issue` in the YAML for historical reasons, but it holds the PR number.

If the user provides a PR description, commit message, or diff, extract what you can from it and
only ask for what is still missing.

## Step 2 — Draft the `body` field

Apply every rule in `.changes/STYLE-GUIDE.md` before writing the body. Key reminders:

- Present tense, active voice
- Lowercase area prefix + colon if scoped to a command (`init:`, `workspace:`, `test:`, etc.)
- Capitalise the first word after the prefix (or at sentence start)
- All identifiers — flags, block names, functions, attributes — in backticks
- `BUG FIXES`: describe the correct behaviour now, not the bug
- `UPGRADE NOTES`: end with a direct call to action ("Review...", "Verify...", "Update...")
- One sentence (two for `UPGRADE NOTES`)

## Step 3 — Self-review the body

Check the draft against the quick checklist in `.changes/STYLE-GUIDE.md` before proceeding. Fix any
failures.

## Step 4 — Create the file

The preferred approach is to use `changie new`, which handles filename generation, directory
placement, and field prompts automatically. Fall back to creating the file directly if `changie`
is not available.

### Option A — Using `changie` (preferred)

If `changie` is not already installed, use one of these methods:

- **macOS (Homebrew):** `brew install changie`
- **Any platform (Go):** `go install github.com/miniscruff/changie@latest`
- **Other platforms:** See the [full installation guide](https://changie.dev/guide/installation/)

`changie` is configured via `.changie.yaml` and writes new entries to `.changes/v1.16/`
automatically.

Run the interactive command and respond to each prompt:

```
changie new
```

Prompts and how to answer them:

| Prompt      | Answer                                    |
| ----------- | ----------------------------------------- |
| `Kind`      | Select the appropriate kind from the menu |
| `Body`      | Paste the drafted body text from Step 2   |
| `PR Number` | Enter the numeric PR number               |

`changie` generates the filename (including the timestamp) and writes the file. Confirm the
written path and show the user the `body` value for verification.

### Option B — Creating the file directly

If `changie` is not available, create `.changes/v1.16/<KIND>-<YYYYMMDD-HHmmss>.yaml` manually
using the current local time for the timestamp.

Multi-word kinds include a space in the filename: `BUG FIXES-20260401-152120.yaml`.

File structure:

```yaml
kind: <KIND>
body: "<body text>"
time: <RFC3339 timestamp with timezone offset>
custom:
  Issue: "<PR number as string>"
```

Note: the `custom.Issue` key holds the PR number despite its name — this is a known quirk of the
project's changie configuration.

## Step 5 — Confirm

Show the user the file path and the final `body` value so they can verify the wording before
committing.
