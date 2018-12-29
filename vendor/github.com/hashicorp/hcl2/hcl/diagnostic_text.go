package hcl

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	wordwrap "github.com/mitchellh/go-wordwrap"
)

type diagnosticTextWriter struct {
	files map[string]*File
	wr    io.Writer
	width uint
	color bool
}

// NewDiagnosticTextWriter creates a DiagnosticWriter that writes diagnostics
// to the given writer as formatted text.
//
// It is designed to produce text appropriate to print in a monospaced font
// in a terminal of a particular width, or optionally with no width limit.
//
// The given width may be zero to disable word-wrapping of the detail text
// and truncation of source code snippets.
//
// If color is set to true, the output will include VT100 escape sequences to
// color-code the severity indicators. It is suggested to turn this off if
// the target writer is not a terminal.
func NewDiagnosticTextWriter(wr io.Writer, files map[string]*File, width uint, color bool) DiagnosticWriter {
	return &diagnosticTextWriter{
		files: files,
		wr:    wr,
		width: width,
		color: color,
	}
}

func (w *diagnosticTextWriter) WriteDiagnostic(diag *Diagnostic) error {
	if diag == nil {
		return errors.New("nil diagnostic")
	}

	var colorCode, highlightCode, resetCode string
	if w.color {
		switch diag.Severity {
		case DiagError:
			colorCode = "\x1b[31m"
		case DiagWarning:
			colorCode = "\x1b[33m"
		}
		resetCode = "\x1b[0m"
		highlightCode = "\x1b[1;4m"
	}

	var severityStr string
	switch diag.Severity {
	case DiagError:
		severityStr = "Error"
	case DiagWarning:
		severityStr = "Warning"
	default:
		// should never happen
		severityStr = "???????"
	}

	fmt.Fprintf(w.wr, "%s%s%s: %s\n\n", colorCode, severityStr, resetCode, diag.Summary)

	if diag.Subject != nil {
		snipRange := *diag.Subject
		highlightRange := snipRange
		if diag.Context != nil {
			// Show enough of the source code to include both the subject
			// and context ranges, which overlap in all reasonable
			// situations.
			snipRange = RangeOver(snipRange, *diag.Context)
		}
		// We can't illustrate an empty range, so we'll turn such ranges into
		// single-character ranges, which might not be totally valid (may point
		// off the end of a line, or off the end of the file) but are good
		// enough for the bounds checks we do below.
		if snipRange.Empty() {
			snipRange.End.Byte++
			snipRange.End.Column++
		}
		if highlightRange.Empty() {
			highlightRange.End.Byte++
			highlightRange.End.Column++
		}

		file := w.files[diag.Subject.Filename]
		if file == nil || file.Bytes == nil {
			fmt.Fprintf(w.wr, "  on %s line %d:\n  (source code not available)\n\n", diag.Subject.Filename, diag.Subject.Start.Line)
		} else {

			var contextLine string
			if diag.Subject != nil {
				contextLine = contextString(file, diag.Subject.Start.Byte)
				if contextLine != "" {
					contextLine = ", in " + contextLine
				}
			}

			fmt.Fprintf(w.wr, "  on %s line %d%s:\n", diag.Subject.Filename, diag.Subject.Start.Line, contextLine)

			src := file.Bytes
			sc := NewRangeScanner(src, diag.Subject.Filename, bufio.ScanLines)

			for sc.Scan() {
				lineRange := sc.Range()
				if !lineRange.Overlaps(snipRange) {
					continue
				}

				beforeRange, highlightedRange, afterRange := lineRange.PartitionAround(highlightRange)
				if highlightedRange.Empty() {
					fmt.Fprintf(w.wr, "%4d: %s\n", lineRange.Start.Line, sc.Bytes())
				} else {
					before := beforeRange.SliceBytes(src)
					highlighted := highlightedRange.SliceBytes(src)
					after := afterRange.SliceBytes(src)
					fmt.Fprintf(
						w.wr, "%4d: %s%s%s%s%s\n",
						lineRange.Start.Line,
						before,
						highlightCode, highlighted, resetCode,
						after,
					)
				}

			}

			w.wr.Write([]byte{'\n'})
		}
	}

	if diag.Detail != "" {
		detail := diag.Detail
		if w.width != 0 {
			detail = wordwrap.WrapString(detail, w.width)
		}
		fmt.Fprintf(w.wr, "%s\n\n", detail)
	}

	return nil
}

func (w *diagnosticTextWriter) WriteDiagnostics(diags Diagnostics) error {
	for _, diag := range diags {
		err := w.WriteDiagnostic(diag)
		if err != nil {
			return err
		}
	}
	return nil
}

func contextString(file *File, offset int) string {
	type contextStringer interface {
		ContextString(offset int) string
	}

	if cser, ok := file.Nav.(contextStringer); ok {
		return cser.ContextString(offset)
	}
	return ""
}
