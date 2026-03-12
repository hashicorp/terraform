// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ast

import (
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type AST struct {
	Nodes []Node
}

type Node interface {
	Children() []Node
	SourceRange() hcl.Range
	String() string
}

func FromConfig(files map[string]*hcl.File) (*AST, tfdiags.Diagnostics) {
	return &AST{}, nil
}

func WriteAST(ast *AST) (map[string][]byte, tfdiags.Diagnostics) {
	// Iterate through top-level nodes in the AST and write them back to their
	// respective files.
	//
	// Top-level nodes include their entire content in String(), so we only need
	// their SourceRange to:
	// - decide which file they belong to
	// - determine how many whitespace lines/columns to insert between them
	var diags tfdiags.Diagnostics
	if ast == nil || len(ast.Nodes) == 0 {
		return map[string][]byte{}, diags
	}

	byFile := map[string][]Node{}
	for _, n := range ast.Nodes {
		if n == nil {
			continue
		}
		rng := n.SourceRange()
		if rng.Filename == "" {
			// Can't place this node into an output file.
			continue
		}
		byFile[rng.Filename] = append(byFile[rng.Filename], n)
	}

	if len(byFile) == 0 {
		return map[string][]byte{}, diags
	}

	out := make(map[string][]byte, len(byFile))

	// Ensure deterministic output ordering by iterating filenames sorted.
	filenames := make([]string, 0, len(byFile))
	for fn := range byFile {
		filenames = append(filenames, fn)
	}
	sort.Strings(filenames)

	for _, filename := range filenames {
		nodes := byFile[filename]
		if len(nodes) == 0 {
			out[filename] = nil
			continue
		}

		// Sort by start position (line then column). If ties, sort by end position.
		sort.SliceStable(nodes, func(i, j int) bool {
			ai, aj := nodes[i].SourceRange(), nodes[j].SourceRange()
			if ai.Start.Line != aj.Start.Line {
				return ai.Start.Line < aj.Start.Line
			}
			if ai.Start.Column != aj.Start.Column {
				return ai.Start.Column < aj.Start.Column
			}
			if ai.End.Line != aj.End.Line {
				return ai.End.Line < aj.End.Line
			}
			return ai.End.Column < aj.End.Column
		})

		var b strings.Builder

		// Pad any initial leading empty lines/columns before the first node.
		prev := nodes[0].SourceRange()
		if prev.Start.Line > 1 {
			b.WriteString(strings.Repeat("\n", prev.Start.Line-1))
		}
		if prev.Start.Column > 1 {
			b.WriteString(strings.Repeat(" ", prev.Start.Column-1))
		}
		b.WriteString(nodes[0].String())

		for i := 1; i < len(nodes); i++ {
			cur := nodes[i].SourceRange()

			lineGap := cur.Start.Line - prev.End.Line
			switch {
			case lineGap > 0:
				b.WriteString(strings.Repeat("\n", lineGap))
				if cur.Start.Column > 1 {
					b.WriteString(strings.Repeat(" ", cur.Start.Column-1))
				}
			case lineGap == 0:
				colGap := cur.Start.Column - prev.End.Column
				if colGap > 0 {
					b.WriteString(strings.Repeat(" ", colGap))
				} else {
					// Overlap or adjacency with unknown intended separator.
					b.WriteString("\n")
					if cur.Start.Column > 1 {
						b.WriteString(strings.Repeat(" ", cur.Start.Column-1))
					}
				}
			default:
				// Current begins before previous ends; fall back to newline separation.
				b.WriteString("\n")
				if cur.Start.Column > 1 {
					b.WriteString(strings.Repeat(" ", cur.Start.Column-1))
				}
			}

			b.WriteString(nodes[i].String())
			prev = cur
		}

		out[filename] = []byte(b.String())
	}

	return out, diags
}
