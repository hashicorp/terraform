package format

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcled"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/mitchellh/colorstring"
	wordwrap "github.com/mitchellh/go-wordwrap"
	"github.com/zclconf/go-cty/cty"
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
			fmt.Fprintf(&buf, "  on %s line %d:\n  (source code not available)\n", highlightRange.Filename, highlightRange.Start.Line)
		} else {
			file, offset := parseRange(src, highlightRange)

			headerRange := highlightRange

			contextStr := hcled.ContextString(file, offset-1)
			if contextStr != "" {
				contextStr = ", in " + contextStr
			}

			fmt.Fprintf(&buf, "  on %s line %d%s:\n", headerRange.Filename, headerRange.Start.Line, contextStr)

			// Config snippet rendering
			sc := hcl.NewRangeScanner(src, highlightRange.Filename, bufio.ScanLines)
			for sc.Scan() {
				lineRange := sc.Range()
				if !lineRange.Overlaps(snippetRange) {
					continue
				}
				beforeRange, highlightedRange, afterRange := lineRange.PartitionAround(highlightRange)
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

		if fromExpr := diag.FromExpr(); fromExpr != nil {
			// We may also be able to generate information about the dynamic
			// values of relevant variables at the point of evaluation, then.
			// This is particularly useful for expressions that get evaluated
			// multiple times with different values, such as blocks using
			// "count" and "for_each", or within "for" expressions.
			expr := fromExpr.Expression
			ctx := fromExpr.EvalContext
			vars := expr.Variables()
			stmts := make([]string, 0, len(vars))
			seen := make(map[string]struct{}, len(vars))
		Traversals:
			for _, traversal := range vars {
				for len(traversal) > 1 {
					val, diags := traversal.TraverseAbs(ctx)
					if diags.HasErrors() {
						// Skip anything that generates errors, since we probably
						// already have the same error in our diagnostics set
						// already.
						traversal = traversal[:len(traversal)-1]
						continue
					}

					traversalStr := traversalStr(traversal)
					if _, exists := seen[traversalStr]; exists {
						continue Traversals // don't show duplicates when the same variable is referenced multiple times
					}
					switch {
					case !val.IsKnown():
						// Can't say anything about this yet, then.
						continue Traversals
					case val.IsNull():
						stmts = append(stmts, fmt.Sprintf(color.Color("[bold]%s[reset] is null"), traversalStr))
					default:
						stmts = append(stmts, fmt.Sprintf(color.Color("[bold]%s[reset] is %s"), traversalStr, compactValueStr(val)))
					}
					seen[traversalStr] = struct{}{}
				}
			}

			sort.Strings(stmts) // FIXME: Should maybe use a traversal-aware sort that can sort numeric indexes properly?

			if len(stmts) > 0 {
				fmt.Fprint(&buf, color.Color("    [dark_gray]|----------------[reset]\n"))
			}
			for _, stmt := range stmts {
				fmt.Fprintf(&buf, color.Color("    [dark_gray]|[reset] %s\n"), stmt)
			}
		}

		buf.WriteByte('\n')
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

func parseRange(src []byte, rng hcl.Range) (*hcl.File, int) {
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
		return file, offset
	}

	return file, offset
}

// traversalStr produces a representation of an HCL traversal that is compact,
// resembles HCL native syntax, and is suitable for display in the UI.
func traversalStr(traversal hcl.Traversal) string {
	// This is a specialized subset of traversal rendering tailored to
	// producing helpful contextual messages in diagnostics. It is not
	// comprehensive nor intended to be used for other purposes.

	var buf bytes.Buffer
	for _, step := range traversal {
		switch tStep := step.(type) {
		case hcl.TraverseRoot:
			buf.WriteString(tStep.Name)
		case hcl.TraverseAttr:
			buf.WriteByte('.')
			buf.WriteString(tStep.Name)
		case hcl.TraverseIndex:
			buf.WriteByte('[')
			if keyTy := tStep.Key.Type(); keyTy.IsPrimitiveType() {
				buf.WriteString(compactValueStr(tStep.Key))
			} else {
				// We'll just use a placeholder for more complex values,
				// since otherwise our result could grow ridiculously long.
				buf.WriteString("...")
			}
			buf.WriteByte(']')
		}
	}
	return buf.String()
}

// compactValueStr produces a compact, single-line summary of a given value
// that is suitable for display in the UI.
//
// For primitives it returns a full representation, while for more complex
// types it instead summarizes the type, size, etc to produce something
// that is hopefully still somewhat useful but not as verbose as a rendering
// of the entire data structure.
func compactValueStr(val cty.Value) string {
	// This is a specialized subset of value rendering tailored to producing
	// helpful but concise messages in diagnostics. It is not comprehensive
	// nor intended to be used for other purposes.

	ty := val.Type()
	switch {
	case val.IsNull():
		return "null"
	case !val.IsKnown():
		// Should never happen here because we should filter before we get
		// in here, but we'll do something reasonable rather than panic.
		return "(not yet known)"
	case ty == cty.Bool:
		if val.True() {
			return "true"
		}
		return "false"
	case ty == cty.Number:
		bf := val.AsBigFloat()
		return bf.Text('g', 10)
	case ty == cty.String:
		// Go string syntax is not exactly the same as HCL native string syntax,
		// but we'll accept the minor edge-cases where this is different here
		// for now, just to get something reasonable here.
		return fmt.Sprintf("%q", val.AsString())
	case ty.IsCollectionType() || ty.IsTupleType():
		l := val.LengthInt()
		switch l {
		case 0:
			return "empty " + ty.FriendlyName()
		case 1:
			return ty.FriendlyName() + " with 1 element"
		default:
			return fmt.Sprintf("%s with %d elements", ty.FriendlyName(), l)
		}
	case ty.IsObjectType():
		atys := ty.AttributeTypes()
		l := len(atys)
		switch l {
		case 0:
			return "object with no attributes"
		case 1:
			var name string
			for k := range atys {
				name = k
			}
			return fmt.Sprintf("object with 1 attribute %q", name)
		default:
			return fmt.Sprintf("object with %d attributes", l)
		}
	default:
		return ty.FriendlyName()
	}
}
