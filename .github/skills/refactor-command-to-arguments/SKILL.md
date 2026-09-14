---
name: refactor-command-to-arguments
description: Use when refactoring a Terraform CLI command in internal/command to use the arguments package to parse and validate flags and arguments supplied to a command.
---

# Refactoring a command to use the arguments package

Terraform CLI commands historically parsed and validated their own flags and arguments directly in their `Run` methods. The
current approach is for each command to use the `internal/command/arguments`
package to parse and validate flags and arguments supplied to the command.
This skill migrates one command at a time.

**Scope of this skill:** the command implementation (`internal/command/<cmd>.go`)
and the view (`internal/command/arguments/<cmd>.go`), plus their tests.

## Reference implementation

Read these before writing any code. They are the canonical example:

- `internal/command/query.go` — an example command that uses the arguments package
- `internal/command/arguments/query.go` — argument parsing for the `query` command
- `internal/command/arguments/query_test.go` — arguments parser tests

Contrast with `internal/command/unlock.go`, which is still parsing CLI flags and arguments in the `Run` method. Note: If you are refactoring this command, propose an updated counter example for this skill. 

## Procedure

### 1. Survey the command before changing anything

Before refactoring the command, check that the command doesn't already use the `arguments` package for parsing and validating its flags and arguments. If that's the case, alert the user and do not proceed with the refactoring.

If the command does not already use the `arguments` package, review the command for existing flag and argument parsing logic that will need to be migrated to the new argument parser. Take note of any default values, required flags, and validation rules that must be preserved in the new parser. Present these rules to the user and ask for them to either approve or modify them before proceeding with the refactoring.

### 2. Write the argument parser

For the given command, create a file at `internal/command/arguments/<cmd>.go`. This file should contain:
- A struct definition that describes the command's expected arguments and flags.
- A function that parses the CLI arguments and flags into an instance of the struct, performing any necessary validation. The function should return both the populated struct and any diagnostics encountered during parsing.
    - The function name should be `Parse<Cmd>`, where `<Cmd>` is the capitalized name of the command. For example: `ParseInit` for the `init` command, and `ParseWorkspaceList` for the `workspace list` command.

The parsing function must produce a best effort interpretation of the arguments even if errors are encountered during parsing. For example, if a user performed a command with the `-json` flag and an invalid flag, the returned struct should still reflect the `-json` flag, and the diagnostics should indicate the invalid flag.

### 3. Rewrite the command's `Run`

Once the arguments parser is implemented, rewrite the command's `Run` method to use the new parser. This typically involves:
- Calling the parsing function at the beginning of the `Run` method.
- Handling any diagnostics returned by the parser, such as displaying error messages to the user.
- Using the populated struct to access the command's arguments and flags, instead of directly reading from the CLI context.

If the command uses `cli.Ui` for displaying messages to the user, ensure that any diagnostics or error messages produced by the argument parser are appropriately communicated through the `cli.Ui` interface. If the command instead uses the `views` package for rendering output, ensure that any diagnostics or error messages are appropriately communicated through the views system.

### 4. Tests

Tests should be created in the `arguments` package, including testing how valid and invalid values are parsed.

These are the types of tests that should always be included:
- Split parser test cases into `TestParse<Cmd>_valid` and
  `TestParse<Cmd>_invalid` tests, following the convention used throughout the
  `arguments` package.
- Handling an unexpected argument.
- Handling an unexpected flag.
- A test that leaves all values at their defaults.
- A test that sets all values to non-default values.
- Tests validating that errors are raised when mutually exclusive flags are supplied together.
- Tests demonstrating any validation logic implemented in the argument parser, e.g. ensuring required flags are provided, or that flag values fall within acceptable ranges.
- If the command accepts flags that enable selecting a `ViewType` that isn't the default human type, e.g. `-json`, include a test showing that the correct view type is selected when the flag is used.

Depending on a command's complexity, there may be other types of test that aren't listed above.

Do not introduce new validation during a behavior-preserving refactor. If the
existing command accepts an otherwise questionable argument combination,
preserve that behavior and add a code comment at the relevant parser location
describing the proposed validation and explicitly noting that it would be a
breaking change. Add a test in the valid cases if needed to lock in the
preserved behavior.

### 5. Verify

Run `gofmt` on every changed Go file, then run the focused parser tests and the
existing command tests:

```text
go test ./internal/command/arguments -run 'TestParse<Cmd>'
go test ./internal/command -run 'Test<Cmd>'
```

Also run `git diff --check` and review the diff to confirm that:

- all pre-existing defaults, flag behavior, positional-argument behavior, and
  exit statuses are preserved;
- no new validation has been introduced; potential stricter validation is
  documented in code with its breaking-change impact;
- parser diagnostics are surfaced by the command;
- the parser returns the best-effort argument values when parsing fails; and
- no command-specific flag parsing remains in `Run`.

## Checklist before reporting done

- [ ] The command did not already use the arguments package, or that was
  confirmed before starting.
- [ ] The approved defaults, required values, and validation rules are
  represented in `internal/command/arguments/<cmd>.go`.
- [ ] `Parse<Cmd>` returns a populated argument struct and diagnostics,
  including best-effort values when parsing reports errors.
- [ ] `Run` calls the parser, surfaces parser diagnostics, and uses the parsed
  struct rather than parsing flags itself.
- [ ] Argument parser tests cover defaults, non-default values, unexpected
  flags and arguments, and command-specific validation or mutually exclusive
  flags where applicable, without adding validation that changes existing
  behavior.
- [ ] Existing command tests and focused parser tests pass.
- [ ] Formatting, `git diff --check`, and the final diff review pass.