// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// View is the base layer for command views, encapsulating a set of I/O
// streams, a colorize implementation, and implementing a human friendly view
// for diagnostics.
type View struct {
	streams  *terminal.Streams
	colorize *colorstring.Colorize

	compactWarnings bool

	// When this is true it's a hint that Terraform is being run indirectly
	// via a wrapper script or other automation and so we may wish to replace
	// direct examples of commands to run with more conceptual directions.
	// However, we only do this on a best-effort basis, typically prioritizing
	// the messages that users are most likely to see.
	runningInAutomation bool

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

// SetRunningInAutomation modifies the view's "running in automation" flag,
// which causes some slight adjustments to certain messages that would normally
// suggest specific Terraform commands to run, to make more conceptual gestures
// instead for situations where the user isn't running Terraform directly.
//
// For convenient use during initialization (in conjunction with NewView),
// SetRunningInAutomation returns the reciever after modifying it.
func (v *View) SetRunningInAutomation(new bool) *View {
	v.runningInAutomation = new
	return v
}

func (v *View) RunningInAutomation() bool {
	return v.runningInAutomation
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
		msg := v.formatDiagnostic(diag)

		if diag.Severity() == tfdiags.Error {
			v.streams.Eprint(msg)
		} else {
			v.streams.Print(msg)
		}
	}
}

// PolicyDiagnostics renders policy diagnostics that are not tied to a
// specific target, such as setup diagnostics (e.g. a failure to connect to
// the policy engine).
func (v *View) PolicyDiagnostics(diags policy.Diagnostics) {
	v.Diagnostics(diags.AsTerraformDiags())
}

func (v *View) PolicyResult(addr string, resp policy.EvaluationResponse, rng hcl.Range) {
	configSources := v.configSources()
	var buf strings.Builder
	var foundInfo bool

	for _, enforcement := range resp.Enforcements {
		var src []byte
		if enforcement.LocalRange != nil {
			src = configSources[enforcement.LocalRange.Filename]
		}
		info := json.NewPolicyInfo(src, enforcement)
		// Print info message attached to the enforcement
		if info.Message != "" {
			foundInfo = true
			buf.WriteString("Policy Info:\n")
			if info.PolicyRange != nil && info.PolicySnippet != nil {
				fmt.Fprintf(
					&buf,
					"on %s line %d, in %s\n",
					info.PolicyRange.Filename,
					info.PolicyRange.Start.Line,
					info.PolicySnippet.Code,
				)
			} else if enforcement.Policy != nil {
				fmt.Fprintf(
					&buf,
					"in policy %s\n",
					enforcement.Policy.Address,
				)
			}
			fmt.Fprintf(&buf, "%q\n", info.Message)

			if !rng.Empty() {
				resourceContext := string(rng.SliceBytes(configSources[rng.Filename]))

				fmt.Fprintf(
					&buf,
					"\non %s line %d, in %s\n",
					rng.Filename,
					rng.Start.Line,
					resourceContext,
				)
			}
			buf.WriteString("\n")
		}
	}

	// Print policy diagnostics
	v.Diagnostics(resp.Diagnostics.AsTerraformDiags())

	if foundInfo {
		v.streams.Println()
		v.streams.Println(buf.String())
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

func (v *View) formatDiagnostic(diag tfdiags.Diagnostic) string {
	if v.colorize.Disable {
		return format.DiagnosticPlain(diag, v.configSources(), v.streams.Stderr.Columns())
	} else {
		return format.Diagnostic(diag, v.configSources(), v.colorize, v.streams.Stderr.Columns())
	}
}
