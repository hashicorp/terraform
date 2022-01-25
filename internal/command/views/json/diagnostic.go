package json

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcled"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// These severities map to the tfdiags.Severity values, plus an explicit
// unknown in case that enum grows without us noticing here.
const (
	DiagnosticSeverityUnknown = "unknown"
	DiagnosticSeverityError   = "error"
	DiagnosticSeverityWarning = "warning"
)

// Diagnostic represents any tfdiags.Diagnostic value. The simplest form has
// just a severity, single line summary, and optional detail. If there is more
// information about the source of the diagnostic, this is represented in the
// range field.
type Diagnostic struct {
	Severity string             `json:"severity"`
	Summary  string             `json:"summary"`
	Detail   string             `json:"detail"`
	Address  string             `json:"address,omitempty"`
	Range    *DiagnosticRange   `json:"range,omitempty"`
	Snippet  *DiagnosticSnippet `json:"snippet,omitempty"`
}

// Pos represents a position in the source code.
type Pos struct {
	// Line is a one-based count for the line in the indicated file.
	Line int `json:"line"`

	// Column is a one-based count of Unicode characters from the start of the line.
	Column int `json:"column"`

	// Byte is a zero-based offset into the indicated file.
	Byte int `json:"byte"`
}

// DiagnosticRange represents the filename and position of the diagnostic
// subject. This defines the range of the source to be highlighted in the
// output. Note that the snippet may include additional surrounding source code
// if the diagnostic has a context range.
//
// The Start position is inclusive, and the End position is exclusive. Exact
// positions are intended for highlighting for human interpretation only and
// are subject to change.
type DiagnosticRange struct {
	Filename string `json:"filename"`
	Start    Pos    `json:"start"`
	End      Pos    `json:"end"`
}

// DiagnosticSnippet represents source code information about the diagnostic.
// It is possible for a diagnostic to have a source (and therefore a range) but
// no source code can be found. In this case, the range field will be present and
// the snippet field will not.
type DiagnosticSnippet struct {
	// Context is derived from HCL's hcled.ContextString output. This gives a
	// high-level summary of the root context of the diagnostic: for example,
	// the resource block in which an expression causes an error.
	Context *string `json:"context"`

	// Code is a possibly-multi-line string of Terraform configuration, which
	// includes both the diagnostic source and any relevant context as defined
	// by the diagnostic.
	Code string `json:"code"`

	// StartLine is the line number in the source file for the first line of
	// the snippet code block. This is not necessarily the same as the value of
	// Range.Start.Line, as it is possible to have zero or more lines of
	// context source code before the diagnostic range starts.
	StartLine int `json:"start_line"`

	// HighlightStartOffset is the character offset into Code at which the
	// diagnostic source range starts, which ought to be highlighted as such by
	// the consumer of this data.
	HighlightStartOffset int `json:"highlight_start_offset"`

	// HighlightEndOffset is the character offset into Code at which the
	// diagnostic source range ends.
	HighlightEndOffset int `json:"highlight_end_offset"`

	// Values is a sorted slice of expression values which may be useful in
	// understanding the source of an error in a complex expression.
	Values []DiagnosticExpressionValue `json:"values"`
}

// DiagnosticExpressionValue represents an HCL traversal string (e.g.
// "var.foo") and a statement about its value while the expression was
// evaluated (e.g. "is a string", "will be known only after apply"). These are
// intended to help the consumer diagnose why an expression caused a diagnostic
// to be emitted.
type DiagnosticExpressionValue struct {
	Traversal string `json:"traversal"`
	Statement string `json:"statement"`
}

// NewDiagnostic takes a tfdiags.Diagnostic and a map of configuration sources,
// and returns a Diagnostic struct.
func NewDiagnostic(diag tfdiags.Diagnostic, sources map[string][]byte) *Diagnostic {
	var sev string
	switch diag.Severity() {
	case tfdiags.Error:
		sev = DiagnosticSeverityError
	case tfdiags.Warning:
		sev = DiagnosticSeverityWarning
	default:
		sev = DiagnosticSeverityUnknown
	}

	desc := diag.Description()

	diagnostic := &Diagnostic{
		Severity: sev,
		Summary:  desc.Summary,
		Detail:   desc.Detail,
		Address:  desc.Address,
	}

	sourceRefs := diag.Source()
	if sourceRefs.Subject != nil {
		// We'll borrow HCL's range implementation here, because it has some
		// handy features to help us produce a nice source code snippet.
		highlightRange := sourceRefs.Subject.ToHCL()

		// Some diagnostic sources fail to set the end of the subject range.
		if highlightRange.End == (hcl.Pos{}) {
			highlightRange.End = highlightRange.Start
		}

		snippetRange := highlightRange
		if sourceRefs.Context != nil {
			snippetRange = sourceRefs.Context.ToHCL()
		}

		// Make sure the snippet includes the highlight. This should be true
		// for any reasonable diagnostic, but we'll make sure.
		snippetRange = hcl.RangeOver(snippetRange, highlightRange)

		// Empty ranges result in odd diagnostic output, so extend the end to
		// ensure there's at least one byte in the snippet or highlight.
		if snippetRange.Empty() {
			snippetRange.End.Byte++
			snippetRange.End.Column++
		}
		if highlightRange.Empty() {
			highlightRange.End.Byte++
			highlightRange.End.Column++
		}

		diagnostic.Range = &DiagnosticRange{
			Filename: highlightRange.Filename,
			Start: Pos{
				Line:   highlightRange.Start.Line,
				Column: highlightRange.Start.Column,
				Byte:   highlightRange.Start.Byte,
			},
			End: Pos{
				Line:   highlightRange.End.Line,
				Column: highlightRange.End.Column,
				Byte:   highlightRange.End.Byte,
			},
		}

		var src []byte
		if sources != nil {
			src = sources[highlightRange.Filename]
		}

		// If we have a source file for the diagnostic, we can emit a code
		// snippet.
		if src != nil {
			diagnostic.Snippet = &DiagnosticSnippet{
				StartLine: snippetRange.Start.Line,

				// Ensure that the default Values struct is an empty array, as this
				// makes consuming the JSON structure easier in most languages.
				Values: []DiagnosticExpressionValue{},
			}

			file, offset := parseRange(src, highlightRange)

			// Some diagnostics may have a useful top-level context to add to
			// the code snippet output.
			contextStr := hcled.ContextString(file, offset-1)
			if contextStr != "" {
				diagnostic.Snippet.Context = &contextStr
			}

			// Build the string of the code snippet, tracking at which byte of
			// the file the snippet starts.
			var codeStartByte int
			sc := hcl.NewRangeScanner(src, highlightRange.Filename, bufio.ScanLines)
			var code strings.Builder
			for sc.Scan() {
				lineRange := sc.Range()
				if lineRange.Overlaps(snippetRange) {
					if codeStartByte == 0 && code.Len() == 0 {
						codeStartByte = lineRange.Start.Byte
					}
					code.Write(lineRange.SliceBytes(src))
					code.WriteRune('\n')
				}
			}
			codeStr := strings.TrimSuffix(code.String(), "\n")
			diagnostic.Snippet.Code = codeStr

			// Calculate the start and end byte of the highlight range relative
			// to the code snippet string.
			start := highlightRange.Start.Byte - codeStartByte
			end := start + (highlightRange.End.Byte - highlightRange.Start.Byte)

			// We can end up with some quirky results here in edge cases like
			// when a source range starts or ends at a newline character,
			// so we'll cap the results at the bounds of the highlight range
			// so that consumers of this data don't need to contend with
			// out-of-bounds errors themselves.
			if start < 0 {
				start = 0
			} else if start > len(codeStr) {
				start = len(codeStr)
			}
			if end < 0 {
				end = 0
			} else if end > len(codeStr) {
				end = len(codeStr)
			}

			diagnostic.Snippet.HighlightStartOffset = start
			diagnostic.Snippet.HighlightEndOffset = end

			if fromExpr := diag.FromExpr(); fromExpr != nil {
				// We may also be able to generate information about the dynamic
				// values of relevant variables at the point of evaluation, then.
				// This is particularly useful for expressions that get evaluated
				// multiple times with different values, such as blocks using
				// "count" and "for_each", or within "for" expressions.
				expr := fromExpr.Expression
				ctx := fromExpr.EvalContext
				vars := expr.Variables()
				values := make([]DiagnosticExpressionValue, 0, len(vars))
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
						value := DiagnosticExpressionValue{
							Traversal: traversalStr,
						}
						switch {
						case val.HasMark(marks.Sensitive):
							// We won't say anything at all about sensitive values,
							// because we might give away something that was
							// sensitive about them.
							value.Statement = "has a sensitive value"
						case !val.IsKnown():
							if ty := val.Type(); ty != cty.DynamicPseudoType {
								value.Statement = fmt.Sprintf("is a %s, known only after apply", ty.FriendlyName())
							} else {
								value.Statement = "will be known only after apply"
							}
						default:
							value.Statement = fmt.Sprintf("is %s", compactValueStr(val))
						}
						values = append(values, value)
						seen[traversalStr] = struct{}{}
					}
				}
				sort.Slice(values, func(i, j int) bool {
					return values[i].Traversal < values[j].Traversal
				})
				diagnostic.Snippet.Values = values
			}
		}
	}

	return diagnostic
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

	// Ignore diagnostics here as there is nothing we can do with them.
	if strings.HasSuffix(filename, ".json") {
		file, _ = parser.ParseJSON(src, filename)
	} else {
		file, _ = parser.ParseHCL(src, filename)
	}

	return file, offset
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

	if val.HasMark(marks.Sensitive) {
		// We check this in here just to make sure, but note that the caller
		// of compactValueStr ought to have already checked this and skipped
		// calling into compactValueStr anyway, so this shouldn't actually
		// be reachable.
		return "(sensitive value)"
	}

	// WARNING: We've only checked that the value isn't sensitive _shallowly_
	// here, and so we must never show any element values from complex types
	// in here. However, it's fine to show map keys and attribute names because
	// those are never sensitive in isolation: the entire value would be
	// sensitive in that case.

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
