// Package printer implements printing of AST nodes to HCL format.
package printer

import (
	"bytes"
	"io"
	"text/tabwriter"

	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/hcl/hcl/parser"
)

var DefaultConfig = Config{
	SpacesWidth: 2,
}

// Filter is an interface that allows external programs to alter HCL in-line as
// it's being parsed for printing. This allows specific examples such as adding
// or altering fields based on context-specific needs.
type Filter interface {
	// Filter should modify the supplied ast.File inline, returning an error if
	// for some reason the process fails. This fails the formatter as well.
	//
	// Care should be taken to supply the filters in the order they are intended
	// to be applied when passing them to the slice in Format.
	Filter(*ast.File) error
}

// A Config node controls the output of Fprint.
type Config struct {
	SpacesWidth int // if set, it will use spaces instead of tabs for alignment
}

func (c *Config) Fprint(output io.Writer, node ast.Node) error {
	p := &printer{
		cfg:                *c,
		comments:           make([]*ast.CommentGroup, 0),
		standaloneComments: make([]*ast.CommentGroup, 0),
		// enableTrace:        true,
	}

	p.collectComments(node)

	if _, err := output.Write(p.unindent(p.output(node))); err != nil {
		return err
	}

	// flush tabwriter, if any
	var err error
	if tw, _ := output.(*tabwriter.Writer); tw != nil {
		err = tw.Flush()
	}

	return err
}

// Fprint "pretty-prints" an HCL node to output
// It calls Config.Fprint with default settings.
func Fprint(output io.Writer, node ast.Node) error {
	return DefaultConfig.Fprint(output, node)
}

// Format formats src HCL and returns the result. External programs can supply
// a list of filters to directly alter the formatted output as well.
func Format(src []byte, filters []Filter) ([]byte, error) {
	node, err := parser.Parse(src)
	if err != nil {
		return nil, err
	}

	for _, f := range filters {
		if err := f.Filter(node); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	if err := DefaultConfig.Fprint(&buf, node); err != nil {
		return nil, err
	}

	// Add trailing newline to result
	buf.WriteString("\n")
	return buf.Bytes(), nil
}
