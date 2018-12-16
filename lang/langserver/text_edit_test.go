package langserver

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/lsp"
)

func TestMakeTextEdits(t *testing.T) {
	tests := map[string]struct {
		Old, New string
		Want     []lsp.TextEdit
	}{
		"empty": {
			"",
			"",
			nil,
		},
		"insert into empty": {
			"",
			"hello",
			[]lsp.TextEdit{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 1, Character: 0},
					},
					NewText: "", // Delete the empty line we assume for an empty string
				},
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 0},
					},
					NewText: "hello\n",
				},
			},
		},
		"delete to empty": {
			"hello",
			"",
			[]lsp.TextEdit{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 1, Character: 0},
					},
					NewText: "",
				},
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 0},
					},
					NewText: "\n", // Insert the empty line we assume for an empty string
				},
			},
		},
		"replace everything": {
			"hello",
			"world",
			[]lsp.TextEdit{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 1, Character: 0},
					},
					NewText: "",
				},
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 0, Character: 0},
						End:   lsp.Position{Line: 0, Character: 0},
					},
					NewText: "world\n",
				},
			},
		},
		"insert into middle": {
			"a\nc\n",
			"a\nb\nc\n",
			[]lsp.TextEdit{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 1, Character: 0},
						End:   lsp.Position{Line: 1, Character: 0},
					},
					NewText: "b\n",
				},
			},
		},
		"update similar line": {
			"a\nsimilar\nc\n",
			"a\nsimilarity\nc\n",
			[]lsp.TextEdit{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 1, Character: 0},
						End:   lsp.Position{Line: 1, Character: 7},
					},
					NewText: "similarity", // the whole line's content gets replaced, for simplicity
				},
			},
		},
		"replace dissimilar line": {
			"a\nsimilar\nc\n",
			"a\nzzzzzar\nc\n",
			[]lsp.TextEdit{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 1, Character: 0},
						End:   lsp.Position{Line: 2, Character: 0},
					},
					NewText: "",
				},
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 1, Character: 0},
						End:   lsp.Position{Line: 1, Character: 0},
					},
					NewText: "zzzzzar\n",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			old := []byte(test.Old)
			new := []byte(test.New)
			oldLs := makeSourceLines("old", old)
			newLs := makeSourceLines("new", new)
			got := makeTextEdits(oldLs, newLs, 0.3)
			reverseTextEditSlice(got) // for easier-to-read test cases above

			if !cmp.Equal(got, test.Want) {
				t.Errorf("wrong result\n%s", cmp.Diff(test.Want, got))
			}
		})
	}
}
