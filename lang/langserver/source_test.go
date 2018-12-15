package langserver

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	lsp "github.com/hashicorp/terraform/internal/lsp"
)

func TestPositionConversions(t *testing.T) {
	src := `all_ascii = "eeee"
bmp = "ěěěě"
astral = "𐆓𐆓𐆓𐆓"
grapheme = "👨‍❤️‍👨👨‍❤️‍👨👨‍❤️‍👨👨‍❤️‍👨"
`

	lines := makeSourceLines("", []byte(src))

	// NOTE: This particular test can't deal with situations where the
	// mapping between the representations is non-reversible, such as
	// where an LSP column points into the middle of a grapheme cluster
	// or into the middle of a UTF-16 surrogate pair. It only tests the
	// happy cases where there is a well-defined mapping.
	tests := []struct {
		HCL hcl.Pos
		LSP lsp.Position
	}{
		{
			HCL: hcl.Pos{Line: 1, Column: 1, Byte: 0},
			LSP: lsp.Position{Line: 0, Character: 0},
		},
		{
			HCL: hcl.Pos{Line: 1, Column: 16, Byte: 15},
			LSP: lsp.Position{Line: 0, Character: 15},
		},
		{
			HCL: hcl.Pos{Line: 2, Column: 5, Byte: 23},
			LSP: lsp.Position{Line: 1, Character: 4},
		},
		{
			HCL: hcl.Pos{Line: 2, Column: 10, Byte: 30},
			LSP: lsp.Position{Line: 1, Character: 9},
		},
		{
			HCL: hcl.Pos{Line: 3, Column: 5, Byte: 40},
			LSP: lsp.Position{Line: 2, Character: 4},
		},
		{
			HCL: hcl.Pos{Line: 3, Column: 13, Byte: 54},
			LSP: lsp.Position{Line: 2, Character: 14},
		},
		{
			HCL: hcl.Pos{Line: 4, Column: 5, Byte: 68},
			LSP: lsp.Position{Line: 3, Character: 4},
		},
		{
			HCL: hcl.Pos{Line: 4, Column: 15, Byte: 116},
			LSP: lsp.Position{Line: 3, Character: 28},
		},
	}

	for _, test := range tests {
		var testName string
		switch {
		case test.HCL.Line < 1:
			testName = "before"
		case (test.HCL.Line - 1) >= len(lines):
			testName = "after"
		default:
			l := lines[test.HCL.Line-1]
			spc := bytes.Index(l.content, []byte{' '})
			testName = string(l.content[:spc])
		}
		t.Run(fmt.Sprintf("%s_%d", testName, test.HCL.Column), func(t *testing.T) {
			gotHCL := lines.posLSPToHCL(test.LSP)
			if gotHCL != test.HCL {
				t.Errorf("wrong posLSPToHCL result\ninput: lsp.Position{Line: %.0f, Character: %.0f}\ngot:   %#v\nwant:  %#v", test.LSP.Line, test.LSP.Character, gotHCL, test.HCL)
			}
			gotLSP := lines.posHCLToLSP(test.HCL)
			if gotLSP != test.LSP {
				t.Errorf("wrong posHCLToLSP result\ninput: %#v\ngot:   lsp.Position{Line: %.0f, Character: %.0f}\nwant:  lsp.Position{Line: %.0f, Character: %.0f}", test.HCL, gotLSP.Line, gotLSP.Character, test.LSP.Line, test.LSP.Character)
			}
			gotByte := lines.posLSPToByte(test.LSP)
			if gotByte != test.HCL.Byte {
				t.Errorf("wrong posLSPToByte result\ninput: lsp.Position{Line: %.0f, Character: %.0f}\ngot:   %#v\nwant:  %#v", test.LSP.Line, test.LSP.Character, gotByte, test.HCL.Byte)
			}
		})
	}
}
