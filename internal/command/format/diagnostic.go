package format

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"

	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/tfdiags"

	"github.com/mitchellh/colorstring"
	wordwrap "github.com/mitchellh/go-wordwrap"
)

var disabledColorize = &colorstring.Colorize{
	Colors:  colorstring.DefaultColors,
	Disable: true,
}

// Diagnostic formats a single diagnostic message.
//
// The width argument specifies at what column the diagnostic messages will
// be wrapped. If set to zero, messages will not be wrapped by this function
// at all. Although the long-form text parts of the message are wrapped,
// not all aspects of the message are guaranteed to fit within the specified
// terminal width.
func Diagnostic(diag tfdiags.Diagnostic, sources map[string][]byte, color *colorstring.Colorize, width int) string {
	return DiagnosticFromJSON(viewsjson.NewDiagnostic(diag, sources), color, width)
}

func DiagnosticFromJSON(diag *viewsjson.Diagnostic, color *colorstring.Colorize, width int) string {
	if diag == nil {
		// No good reason to pass a nil diagnostic in here...
		return ""
	}

	var buf bytes.Buffer

	// these leftRule* variables are markers for the beginning of the lines
	// containing the diagnostic that are intended to help sighted users
	// better understand the information hierarchy when diagnostics appear
	// alongside other information or alongside other diagnostics.
	//
	// Without this, it seems (based on folks sharing incomplete messages when
	// asking questions, or including extra content that's not part of the
	// diagnostic) that some readers have trouble easily identifying which
	// text belongs to the diagnostic and which does not.
	var leftRuleLine, leftRuleStart, leftRuleEnd string
	var leftRuleWidth int // in visual character cells

	switch diag.Severity {
	case viewsjson.DiagnosticSeverityError:
		buf.WriteString(color.Color("[bold][red]Error: [reset]"))
		leftRuleLine = color.Color("[red]│[reset] ")
		leftRuleStart = color.Color("[red]╷[reset]")
		leftRuleEnd = color.Color("[red]╵[reset]")
		leftRuleWidth = 2
	case viewsjson.DiagnosticSeverityWarning:
		buf.WriteString(color.Color("[bold][yellow]Warning: [reset]"))
		leftRuleLine = color.Color("[yellow]│[reset] ")
		leftRuleStart = color.Color("[yellow]╷[reset]")
		leftRuleEnd = color.Color("[yellow]╵[reset]")
		leftRuleWidth = 2
	default:
		// Clear out any coloring that might be applied by Terraform's UI helper,
		// so our result is not context-sensitive.
		buf.WriteString(color.Color("\n[reset]"))
	}

	// We don't wrap the summary, since we expect it to be terse, and since
	// this is where we put the text of a native Go error it may not always
	// be pure text that lends itself well to word-wrapping.
	fmt.Fprintf(&buf, color.Color("[bold]%s[reset]\n\n"), diag.Summary)

	appendSourceSnippets(&buf, diag, color)

	if diag.Detail != "" {
		paraWidth := width - leftRuleWidth - 1 // leave room for the left rule
		if paraWidth > 0 {
			lines := strings.Split(diag.Detail, "\n")
			for _, line := range lines {
				if !strings.HasPrefix(line, " ") {
					line = wordwrap.WrapString(line, uint(paraWidth))
				}
				fmt.Fprintf(&buf, "%s\n", line)
			}
		} else {
			fmt.Fprintf(&buf, "%s\n", diag.Detail)
		}
	}

	// Before we return, we'll finally add the left rule prefixes to each
	// line so that the overall message is visually delimited from what's
	// around it. We'll do that by scanning over what we already generated
	// and adding the prefix for each line.
	var ruleBuf strings.Builder
	sc := bufio.NewScanner(&buf)
	ruleBuf.WriteString(leftRuleStart)
	ruleBuf.WriteByte('\n')
	for sc.Scan() {
		line := sc.Text()
		prefix := leftRuleLine
		if line == "" {
			// Don't print the space after the line if there would be nothing
			// after it anyway.
			prefix = strings.TrimSpace(prefix)
		}
		ruleBuf.WriteString(prefix)
		ruleBuf.WriteString(line)
		ruleBuf.WriteByte('\n')
	}
	ruleBuf.WriteString(leftRuleEnd)
	ruleBuf.WriteByte('\n')

	return ruleBuf.String()
}

// DiagnosticPlain is an alternative to Diagnostic which minimises the use of
// virtual terminal formatting sequences.
//
// It is intended for use in automation and other contexts in which diagnostic
// messages are parsed from the Terraform output.
func DiagnosticPlain(diag tfdiags.Diagnostic, sources map[string][]byte, width int) string {
	return DiagnosticPlainFromJSON(viewsjson.NewDiagnostic(diag, sources), width)
}

func DiagnosticPlainFromJSON(diag *viewsjson.Diagnostic, width int) string {
	if diag == nil {
		// No good reason to pass a nil diagnostic in here...
		return ""
	}

	var buf bytes.Buffer

	switch diag.Severity {
	case viewsjson.DiagnosticSeverityError:
		buf.WriteString("\nError: ")
	case viewsjson.DiagnosticSeverityWarning:
		buf.WriteString("\nWarning: ")
	default:
		buf.WriteString("\n")
	}

	// We don't wrap the summary, since we expect it to be terse, and since
	// this is where we put the text of a native Go error it may not always
	// be pure text that lends itself well to word-wrapping.
	fmt.Fprintf(&buf, "%s\n\n", diag.Summary)

	appendSourceSnippets(&buf, diag, disabledColorize)

	if diag.Detail != "" {
		if width > 1 {
			lines := strings.Split(diag.Detail, "\n")
			for _, line := range lines {
				if !strings.HasPrefix(line, " ") {
					line = wordwrap.WrapString(line, uint(width-1))
				}
				fmt.Fprintf(&buf, "%s\n", line)
			}
		} else {
			fmt.Fprintf(&buf, "%s\n", diag.Detail)
		}
	}

	return buf.String()
}

// DiagnosticWarningsCompact is an alternative to Diagnostic for when all of
// the given diagnostics are warnings and we want to show them compactly,
// with only two lines per warning and excluding all of the detail information.
//
// The caller may optionally pre-process the given diagnostics with
// ConsolidateWarnings, in which case this function will recognize consolidated
// messages and include an indication that they are consolidated.
//
// Do not pass non-warning diagnostics to this function, or the result will
// be nonsense.
func DiagnosticWarningsCompact(diags tfdiags.Diagnostics, color *colorstring.Colorize) string {
	var b strings.Builder
	b.WriteString(color.Color("[bold][yellow]Warnings:[reset]\n\n"))
	for _, diag := range diags {
		sources := tfdiags.WarningGroupSourceRanges(diag)
		b.WriteString(fmt.Sprintf("- %s\n", diag.Description().Summary))
		if len(sources) > 0 {
			mainSource := sources[0]
			if mainSource.Subject != nil {
				if len(sources) > 1 {
					b.WriteString(fmt.Sprintf(
						"  on %s line %d (and %d more)\n",
						mainSource.Subject.Filename,
						mainSource.Subject.Start.Line,
						len(sources)-1,
					))
				} else {
					b.WriteString(fmt.Sprintf(
						"  on %s line %d\n",
						mainSource.Subject.Filename,
						mainSource.Subject.Start.Line,
					))
				}
			} else if len(sources) > 1 {
				b.WriteString(fmt.Sprintf(
					"  (%d occurences of this warning)\n",
					len(sources),
				))
			}
		}
	}

	return b.String()
}

func appendSourceSnippets(buf *bytes.Buffer, diag *viewsjson.Diagnostic, color *colorstring.Colorize) {
	if diag.Address != "" {
		fmt.Fprintf(buf, "  with %s,\n", diag.Address)
	}

	if diag.Range == nil {
		return
	}

	if diag.Snippet == nil {
		// This should generally not happen, as long as sources are always
		// loaded through the main loader. We may load things in other
		// ways in weird cases, so we'll tolerate it at the expense of
		// a not-so-helpful error message.
		fmt.Fprintf(buf, "  on %s line %d:\n  (source code not available)\n", diag.Range.Filename, diag.Range.Start.Line)
	} else {
		snippet := diag.Snippet
		code := snippet.Code

		var contextStr string
		if snippet.Context != nil {
			contextStr = fmt.Sprintf(", in %s", *snippet.Context)
		}
		fmt.Fprintf(buf, "  on %s line %d%s:\n", diag.Range.Filename, diag.Range.Start.Line, contextStr)

		// Split the snippet and render the highlighted section with underlines
		start := snippet.HighlightStartOffset
		end := snippet.HighlightEndOffset

		// Only buggy diagnostics can have an end range before the start, but
		// we need to ensure we don't crash here if that happens.
		if end < start {
			end = start + 1
			if end > len(code) {
				end = len(code)
			}
		}

		// If either start or end is out of range for the code buffer then
		// we'll cap them at the bounds just to avoid a panic, although
		// this would happen only if there's a bug in the code generating
		// the snippet objects.
		if start < 0 {
			start = 0
		} else if start > len(code) {
			start = len(code)
		}
		if end < 0 {
			end = 0
		} else if end > len(code) {
			end = len(code)
		}

		before, highlight, after := code[0:start], code[start:end], code[end:]
		code = fmt.Sprintf(color.Color("%s[underline]%s[reset]%s"), before, highlight, after)

		// Split the snippet into lines and render one at a time
		lines := strings.Split(code, "\n")
		for i, line := range lines {
			fmt.Fprintf(
				buf, "%4d: %s\n",
				snippet.StartLine+i,
				line,
			)
		}

		if len(snippet.Values) > 0 {
			// The diagnostic may also have information about the dynamic
			// values of relevant variables at the point of evaluation.
			// This is particularly useful for expressions that get evaluated
			// multiple times with different values, such as blocks using
			// "count" and "for_each", or within "for" expressions.
			values := make([]viewsjson.DiagnosticExpressionValue, len(snippet.Values))
			copy(values, snippet.Values)
			sort.Slice(values, func(i, j int) bool {
				return values[i].Traversal < values[j].Traversal
			})

			fmt.Fprint(buf, color.Color("    [dark_gray]├────────────────[reset]\n"))
			for _, value := range values {
				fmt.Fprintf(buf, color.Color("    [dark_gray]│[reset] [bold]%s[reset] %s\n"), value.Traversal, value.Statement)
			}
		}
	}

	buf.WriteByte('\n')
}
