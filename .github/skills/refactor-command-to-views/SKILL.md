---
name: refactor-command-to-views
description: Use when refactoring a Terraform CLI command in internal/command to stop producing terminal output via cli.Ui (c.Ui.Output, c.Ui.Error, c.showDiagnostics) and instead produce output through a command-specific view in internal/command/views. Also use when asked to add JSON (-json) output to an existing command, or to migrate a command off Meta.process.
---

# Refactoring a command to use the views package

Terraform CLI commands historically wrote output directly to a `cli.Ui`. The
current approach is for each command to own an interface in
`internal/command/views`, with one implementation per output format (human,
JSON, ...). This skill migrates one command at a time.

**Scope of this skill:** the command implementation (`internal/command/<cmd>.go`)
and the view (`internal/command/views/<cmd>.go`), plus their tests. The
arguments parser in `internal/command/arguments` is assumed to already exist. If
it does not, stop and tell the user — do not invent one.

## Reference implementation

Read these before writing any code. They are the canonical example:

- `internal/command/workspace_list.go` — the migrated command
- `internal/command/views/workspace_list.go` — the view interface + Human/JSON impls
- `internal/command/views/workspace_list_test.go` — view tests
- `internal/command/arguments/workspace_list.go` — the arguments parser
- `internal/command/views/view.go` — the base `View` type (`Diagnostics`, `streams`, `Configure`)

Contrast with `internal/command/workspace_show.go`, which is still on the old
`cli.Ui` approach.

## Procedure

### 1. Survey the command before changing anything

Establish and write down:

- Every output site: `c.Ui.Output`, `c.Ui.Error`, `c.Ui.Warn`, `c.Ui.Info`,
  `c.showDiagnostics`, direct `fmt.Print*`, and any helper that writes output.
- Every `return` path through `Run`, and what is printed on each.
- Whether the command's arguments parser already returns a `ViewType`
  (check `internal/command/arguments/<cmd>.go` for `ViewHuman` / `ViewJSON`).
- Whether the command currently supports `-json` (check `Help()` and the parser).
- Existing tests: `internal/command/<cmd>_test.go` and any golden output.

**Do not change user-visible human output.** The human view must reproduce the
existing bytes, including blank lines and error prefixes. Golden/CLI tests are
the contract.

### 2. Decide the view interface

The interface method set is driven by the command's output shape, not by the
number of existing `c.Ui` calls. Name methods after what the command *does*
(`List`, `Show`, `New`), not after the format.

Signature rule: each method takes everything needed to render the complete
output for that code path, including `diags tfdiags.Diagnostics` last. The view
— not the command — decides ordering of diagnostics relative to primary output.

```go
// The WorkspaceList view is used for the `workspace list` subcommand.
type WorkspaceList interface {
	List(selected string, list []string, diags tfdiags.Diagnostics)
}
```

**Static JSON output rule.** If the command emits JSON as a single object
(a "static log") rather than a stream of JSON events, then every path through
`Run` must call **exactly one** view method **exactly once**. This is what
guarantees a single well-formed JSON document on stdout. Structure `Run` by
accumulating into a `diags` variable and calling the single view method
immediately before each `return`. Never call a view method and then continue on
to another one.

If the command genuinely needs incremental/streaming JSON, say so explicitly and
model it on the streaming views (e.g. `internal/command/views/json_view.go`)
rather than the static pattern.

### 3. Write the view

Create/extend `internal/command/views/<cmd>.go`:

- The interface, documented with which subcommand it serves.
- A `New<Cmd>(viewType arguments.ViewType, view *View) <Cmd>` constructor that
  switches on the view type and `panic`s on an unsupported type.
- `<Cmd>Human` struct with a `view *View` field, plus
  `var _ <Cmd> = (*<Cmd>Human)(nil)`.
- `<Cmd>JSON` **only if the command already supports `-json`**. Do not add a
  JSON view speculatively; if the user wants new `-json` support, that is a
  separate, larger change (parser, help text, docs, changelog) — confirm first.

Human implementation:

- Render diagnostics with `v.view.Diagnostics(diags)` — never format
  diagnostics by hand.
- Write primary output with `v.view.streams.Print`/`Println`. Do not use
  `fmt.Print*`.
- Keep the ordering the old command had (usually diagnostics first, or errors to
  stderr and output to stdout).

JSON implementation:

- Define an output struct with a `FormatVersion string \`json:"format_version"\``
  field, set to `"1.0"` for a new format, and a
  `Diagnostics []*viewsjson.Diagnostic \`json:"diagnostics"\`` field.
- Convert diagnostics with
  `viewsjson.NewDiagnostic(diag, v.view.configSources())`.
- Normalise `nil` slices to empty slices so they serialise as `[]`, not `null`.
- `json.MarshalIndent(output, "", "  ")`, `panic` on marshal error (input is
  fully controlled), and emit with `v.view.streams.Println`.

Diagnostic constructors that exist only to serve this view (e.g.
`warnNoEnvsExistDiag`) belong in the views package, not the command package, so
both implementations share them.

### 4. Rewrite the command's `Run`

Target shape:

```go
func (c *XCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Parse command-specific arguments.
	args, diags := arguments.ParseX(rawArgs)

	// Prepare the view
	view := views.NewX(args.ViewType, c.View)

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.X(/* zero values */, diags)
		return 1
	}

	// ... work, appending to diags and returning via the single view method ...

	view.X(result, diags)
	return 0
}
```

Rules:

- Remove the `c.Meta.process(rawArgs)` call and replace it with
  `arguments.ParseView` + `c.View.Configure`. `Meta.process` also wraps `c.Ui`
  in a `ColorizeUi` and mirrors config onto the view; after removal, **any
  remaining `c.Ui` use in this command will be uncolored and unconfigured**.
  So all output sites in the command must move to the view in the same change.
  If some shared helper the command calls still writes to `c.Ui`, flag it to the
  user rather than silently leaving mixed output.
- The view must be constructed **before** the first point where diagnostics can
  be emitted, so that argument-parsing errors are rendered in the user's
  requested format. Parsing must therefore not be fatal: the parser returns a
  best-effort value alongside errors.
- **Single accumulative `diags` collection**: Declare `var diags tfdiags.Diagnostics`
  at the top of `Run` and append all subsequent diagnostics to it throughout the
  execution of the command (e.g., `args, parseDiags := ...; diags = diags.Append(parseDiags)`).
  The motivation is to ensure warning diagnostics are never dropped: when an
  error occurs, all accumulated diagnostics (warnings and errors) are rendered
  before exiting (`view.Diagnostics(diags)`), and when operations succeed, warnings
  are preserved and passed down to the final view method call. If a single
  accumulative `diags` variable is not appropriate for a specific command, explicitly
  state the rationale.
- Accumulate with `diags = diags.Append(...)` rather than printing as you go.
- Errors previously reported as `c.Ui.Error(fmt.Sprintf(...))` become
  diagnostics appended to `diags`.
- For `Meta` helper methods that return `tfdiags.Diagnostics` (such as
  `c.Meta.checkRequiredVersion()` or `c.backendFromConfig()`), check
  `diags.HasErrors()` rather than `diags != nil`, and append them to `diags`.
- Do not return `cli.RunResultHelp` from a migrated command if it would bypass
  the view; return `1` after rendering diagnostics. If the existing behaviour
  showed help on bad flags, confirm with the user before dropping it.
- Drop now-unused imports (`fmt`, `github.com/hashicorp/cli`).

### 5. Tests

Both are required.

**View tests** — `internal/command/views/<cmd>_test.go`, modelled on
`workspace_list_test.go`:

```go
streams, done := terminal.StreamsForTesting(t)
view := NewView(streams)
view.Configure(&arguments.View{NoColor: true})
v := NewX(arguments.ViewHuman, view)

v.X(...)

output := done(t)
```

- Assert stdout and stderr **separately**; `output.All()` interleaves them.
- Table-driven, covering at minimum: success, success with warning, and error.
- For the JSON view, unmarshal the output and compare structs (or compare
  against a formatted JSON string), and assert the output is a single valid
  JSON document.

**Command tests** — `internal/command/<cmd>_test.go`: update to read output from
the view's streams rather than a mock `cli.Ui`, keeping existing expected output
byte-identical wherever the human format is unchanged.

- **Ensure `Meta.View` is initialized in all test cases**: Check all test
  fixtures and helper functions in `<cmd>_test.go`. Older tests often construct
  `Meta` with only `Ui: ui` and leave `View` as `nil`. Because `Run` calls
  `c.View.Configure(common)`, an uninitialized `View` will cause a nil pointer
  dereference panic. Ensure `View: view` is provided (e.g. `view, done := testView(t)`).
- **Stream colorization and plain-text matching**: `testView(t)` defaults to
  ANSI colorization enabled. When testing plain string matches against
  `done(t).Stdout()` or `done(t).Stderr()`, either pass `-no-color` in the
  arguments slice or configure the view with `view.Configure(&arguments.View{NoColor: true})`
  to avoid unexpected ANSI escape codes in output assertions.
- **Diagnostic trailing newlines**: `views.View.Diagnostics` formats diagnostics
  directly to the output streams, which can result in slightly different trailing
  newlines compared to legacy `c.showDiagnostics` / `c.Ui.Error` calls. Use
  `strings.TrimSpace(...)` or `cmp.Diff` with trimmed strings when comparing
  diagnostic error/warning outputs.

### 6. Verify

```
go test ./internal/command/views/... -run 'X'
go test ./internal/command/... -run 'X'
```

Then build the binary and run the command by hand in both modes if it supports
`-json`, confirming JSON output parses:

```
go run . -chdir=<dir> <cmd> -json | jq .
```

## Checklist before reporting done

- [ ] Single `diags` variable declared at start of `Run` and accumulated throughout (or explicitly justified).
- [ ] No `c.Ui.*`, `c.showDiagnostics`, or `fmt.Print*` remains in the command.
- [ ] No `c.Meta.process` call remains in the command.
- [ ] View is constructed before any diagnostics can be emitted.
- [ ] Every `return` path renders through the view.
- [ ] For static JSON: exactly one view method call per path.
- [ ] JSON output has `format_version`, and `[]` rather than `null` for slices.
- [ ] Human output is byte-identical to before the refactor.
- [ ] View tests and command tests pass.
