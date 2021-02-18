package views

import (
	"github.com/hashicorp/terraform/command/arguments"
	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/mitchellh/colorstring"
)

// View is the base layer for command views, encapsulating a set of I/O
// streams, a colorize implementation, and implementing a human friendly view
// for diagnostics.
type View struct {
	streams  *terminal.Streams
	colorize *colorstring.Colorize

	// NOTE: compactWarnings is currently always false. When implementing
	// views for commands which support this flag, we will need to address this.
	compactWarnings bool

	// This unfortunate wart is required to enable rendering of diagnostics which
	// have associated source code in the configuration. This function pointer
	// will be dereferenced as late as possible when rendering diagnostics in
	// order to access the config loader cache.
	configSources func() map[string][]byte
}

// Initialize a View with the given streams, a disabled colorize object, and a
// no-op configSources callback.
func NewView(streams *terminal.Streams) *View {
	return &View{
		streams: streams,
		colorize: &colorstring.Colorize{
			Colors:  colorstring.DefaultColors,
			Disable: true,
			Reset:   true,
		},
		configSources: func() map[string][]byte { return nil },
	}
}

// Configure applies the global view configuration flags.
func (v *View) Configure(view *arguments.View) {
	v.colorize.Disable = view.NoColor
	v.compactWarnings = view.CompactWarnings
}

// SetConfigSources overrides the default no-op callback with a new function
// pointer, and should be called when the config loader is initialized.
func (v *View) SetConfigSources(cb func() map[string][]byte) {
	v.configSources = cb
}

// Diagnostics renders a set of warnings and errors in human-readable form.
// Warnings are printed to stdout, and errors to stderr.
func (v *View) Diagnostics(diags tfdiags.Diagnostics) {
	diags.Sort()

	if len(diags) == 0 {
		return
	}

	diags = diags.ConsolidateWarnings(1)

	// Since warning messages are generally competing
	if v.compactWarnings {
		// If the user selected compact warnings and all of the diagnostics are
		// warnings then we'll use a more compact representation of the warnings
		// that only includes their summaries.
		// We show full warnings if there are also errors, because a warning
		// can sometimes serve as good context for a subsequent error.
		useCompact := true
		for _, diag := range diags {
			if diag.Severity() != tfdiags.Warning {
				useCompact = false
				break
			}
		}
		if useCompact {
			msg := format.DiagnosticWarningsCompact(diags, v.colorize)
			msg = "\n" + msg + "\nTo see the full warning notes, run Terraform without -compact-warnings.\n"
			v.streams.Print(msg)
			return
		}
	}

	for _, diag := range diags {
		var msg string
		if v.colorize.Disable {
			msg = format.DiagnosticPlain(diag, v.configSources(), v.streams.Stderr.Columns())
		} else {
			msg = format.Diagnostic(diag, v.configSources(), v.colorize, v.streams.Stderr.Columns())
		}

		if diag.Severity() == tfdiags.Error {
			v.streams.Eprint(msg)
		} else {
			v.streams.Print(msg)
		}
	}
}

// HelpPrompt is intended to be called from commands which fail to parse all
// of their CLI arguments successfully. It refers users to the full help output
// rather than rendering it directly, which can be overwhelming and confusing.
func (v *View) HelpPrompt(command string) {
	v.streams.Eprintf(helpPrompt, command)
}

const helpPrompt = `
For more help on using this command, run:
  terraform %s -help
`

// outputColumns returns the number of text character cells any non-error
// output should be wrapped to.
//
// This is the number of columns to use if you are calling v.streams.Print or
// related functions.
func (v *View) outputColumns() int {
	return v.streams.Stdout.Columns()
}

// errorColumns returns the number of text character cells any error
// output should be wrapped to.
//
// This is the number of columns to use if you are calling v.streams.Eprint
// or related functions.
func (v *View) errorColumns() int {
	return v.streams.Stderr.Columns()
}

// outputHorizRule will call v.streams.Println with enough horizontal line
// characters to fill an entire row of output.
//
// If UI color is enabled, the rule will get a dark grey coloring to try to
// visually de-emphasize it.
func (v *View) outputHorizRule() {
	v.streams.Println(format.HorizontalRule(v.colorize, v.outputColumns()))
}
