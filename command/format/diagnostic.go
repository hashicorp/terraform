package format

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcled"
	"github.com/hashicorp/hcl2/hclparse"
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
func Diagnostic(diag tfdiags.Diagnostic, sources map[string][]byte, color *colorstring.Colorize, width int) string {
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
	fmt.Fprintf(&buf, color.Color("[bold]%s[reset]\n\n"), desc.Summary)

	if sourceRefs.Subject != nil {
		// We'll borrow HCL's range implementation here, because it has some
		// handy features to help us produce a nice source code snippet.
		highlightRange := sourceRefs.Subject.ToHCL()
		snippetRange := highlightRange
		if sourceRefs.Context != nil {
			snippetRange = sourceRefs.Context.ToHCL()
		}
		// Make sure the snippet includes the highlight. This should be true
		// for any reasonable diagnostic, but we'll make sure.
		snippetRange = hcl.RangeOver(snippetRange, highlightRange)

		// We can't illustrate an empty range, so we'll turn such ranges into
		// single-character ranges, which might not be totally valid (may point
		// off the end of a line, or off the end of the file) but are good
		// enough for the bounds checks we do below.
		if snippetRange.Empty() {
			snippetRange.End.Byte++
			snippetRange.End.Column++
		}
		if highlightRange.Empty() {
			highlightRange.End.Byte++
			highlightRange.End.Column++
		}

		var src []byte
		if sources != nil {
			src = sources[snippetRange.Filename]
		}
		if src == nil {
			// This should generally not happen, as long as sources are always
			// loaded through the main loader. We may load things in other
			// ways in weird cases, so we'll tolerate it at the expense of
			// a not-so-helpful error message.
			fmt.Fprintf(&buf, "  on %s line %d:\n  (source code not available)\n\n", highlightRange.Filename, highlightRange.Start.Line)
		} else {
			contextStr := sourceCodeContextStr(src, highlightRange)
			if contextStr != "" {
				contextStr = ", in " + contextStr
			}
			fmt.Fprintf(&buf, "  on %s line %d%s:\n", highlightRange.Filename, highlightRange.Start.Line, contextStr)

			sc := hcl.NewRangeScanner(src, highlightRange.Filename, bufio.ScanLines)
			for sc.Scan() {
				lineRange := sc.Range()
				if !lineRange.Overlaps(snippetRange) {
					continue
				}
				beforeRange, highlightedRange, afterRange := lineRange.PartitionAround(highlightRange)
				if highlightedRange.Empty() {
					fmt.Fprintf(&buf, "%4d: %s\n", lineRange.Start.Line, sc.Bytes())
				} else {
					before := beforeRange.SliceBytes(src)
					highlighted := highlightedRange.SliceBytes(src)
					after := afterRange.SliceBytes(src)
					fmt.Fprintf(
						&buf, color.Color("%4d: %s[underline]%s[reset]%s\n"),
						lineRange.Start.Line,
						before, highlighted, after,
					)
				}
			}

			buf.WriteByte('\n')
		}
	}

	if desc.Detail != "" {
		detail := desc.Detail
		if width != 0 {
			detail = wordwrap.WrapString(detail, uint(width))
		}
		fmt.Fprintf(&buf, "%s\n", detail)
	}

	return buf.String()
}

// sourceCodeContextStr attempts to find a user-friendly description of
// the location of the given range in the given source code.
//
// An empty string is returned if no suitable description is available, e.g.
// because the source is invalid, or because the offset is not inside any sort
// of identifiable container.
func sourceCodeContextStr(src []byte, rng hcl.Range) string {
	filename := rng.Filename
	offset := rng.Start.Byte

	// We need to re-parse here to get a *hcl.File we can interrogate. This
	// is not awesome since we presumably already parsed the file earlier too,
	// but this re-parsing is architecturally simpler than retaining all of
	// the hcl.File objects and we only do this in the case of an error anyway
	// so the overhead here is not a big problem.
	parser := hclparse.NewParser()
	var file *hcl.File
	var diags hcl.Diagnostics
	if strings.HasSuffix(filename, ".json") {
		file, diags = parser.ParseJSON(src, filename)
	} else {
		file, diags = parser.ParseHCL(src, filename)
	}
	if diags.HasErrors() {
		return ""
	}

	return hcled.ContextString(file, offset)
}
