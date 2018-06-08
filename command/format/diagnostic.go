package format

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform/tfdiags"
	"github.com/mitchellh/colorstring"
	wordwrap "github.com/mitchellh/go-wordwrap"
)

// Diagnostic formats a single diagnostic message.
//
// The width argument specifies at what column the diagnostic messages will
// be wrapped. If set to zero, messages will not be wrapped by this function
// at all. Although the long-form text parts of the message are wrapped,
// not all aspects of the message are guaranteed to fit within the specified
// terminal width.
func Diagnostic(diag tfdiags.Diagnostic, color *colorstring.Colorize, width int) string {
	if diag == nil {
		// No good reason to pass a nil diagnostic in here...
		return ""
	}

	var buf bytes.Buffer

	switch diag.Severity() {
	case tfdiags.Error:
		buf.WriteString(color.Color("\n[bold][red]Error: [reset]"))
	case tfdiags.Warning:
		buf.WriteString(color.Color("\n[bold][yellow]Warning: [reset]"))
	default:
		// Clear out any coloring that might be applied by Terraform's UI helper,
		// so our result is not context-sensitive.
		buf.WriteString(color.Color("\n[reset]"))
	}

	desc := diag.Description()
	sourceRefs := diag.Source()

	// We don't wrap the summary, since we expect it to be terse, and since
	// this is where we put the text of a native Go error it may not always
	// be pure text that lends itself well to word-wrapping.
	if sourceRefs.Subject != nil {
		fmt.Fprintf(&buf, color.Color("[bold]%s[reset] at %s\n\n"), desc.Summary, sourceRefs.Subject.StartString())
	} else {
		fmt.Fprintf(&buf, color.Color("[bold]%s[reset]\n\n"), desc.Summary)
	}

	// TODO: also print out the relevant snippet of config source with the
	// relevant section highlighted, so the user doesn't need to manually
	// correlate back to config. Before we can do this, the HCL2 parser
	// needs to be more deeply integrated so that we can use it to obtain
	// the parsed source code and AST.

	if desc.Detail != "" {
		detail := desc.Detail
		if width != 0 {
			detail = wordwrap.WrapString(detail, uint(width))
		}
		fmt.Fprintf(&buf, "%s\n", detail)
	}

	return buf.String()
}
