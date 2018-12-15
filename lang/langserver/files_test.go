package langserver

import (
	"bytes"
	"testing"

	"github.com/hashicorp/terraform/internal/lsp"
)

func TestFileApplyChange(t *testing.T) {
	tests := map[string]struct {
		Input  string
		Change lsp.TextDocumentContentChangeEvent
		Want   string
	}{
		"empty": {
			``,
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 0},
				},
				RangeLength: 0,
				Text:        ``,
			},
			``,
		},
		"insert to empty": {
			``,
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 0},
				},
				RangeLength: 0,
				Text:        "hello",
			},
			`hello`,
		},
		"insert to empty newline": {
			``,
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 0},
				},
				RangeLength: 0,
				Text:        "hello\n",
			},
			"hello\n",
		},
		"delete start of line": {
			`hello`,
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 2},
				},
				RangeLength: 2,
				Text:        "",
			},
			"llo",
		},
		"replace start of line": {
			`hello`,
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 2},
				},
				RangeLength: 2,
				Text:        "zi",
			},
			"zillo",
		},
		"delete end of line": {
			`hello`,
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 4},
					End:   lsp.Position{Line: 0, Character: 5},
				},
				RangeLength: 1,
				Text:        "",
			},
			"hell",
		},
		"replace end of line": {
			`hello`,
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 4},
					End:   lsp.Position{Line: 0, Character: 5},
				},
				RangeLength: 1,
				Text:        " yes\n",
			},
			"hell yes\n",
		},
		"delete middle of line": {
			`hello`,
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 2},
					End:   lsp.Position{Line: 0, Character: 4},
				},
				RangeLength: 2,
				Text:        "",
			},
			"heo",
		},
		"replace middle of line": {
			`hello`,
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 1},
					End:   lsp.Position{Line: 0, Character: 2},
				},
				RangeLength: 2,
				Text:        "u",
			},
			"hullo",
		},
		"replace middle of line newline": {
			`hello`,
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 1},
					End:   lsp.Position{Line: 0, Character: 2},
				},
				RangeLength: 2,
				Text:        "\n",
			},
			"h\nllo",
		},
		"delete newline": {
			"hello\nworld",
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 5},
					End:   lsp.Position{Line: 1, Character: 0},
				},
				RangeLength: 2,
				Text:        "",
			},
			"helloworld",
		},
		"replace newline": {
			"hello\nworld",
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 5},
					End:   lsp.Position{Line: 1, Character: 0},
				},
				RangeLength: 2,
				Text:        ", ",
			},
			"hello, world",
		},
		"replace across newline": {
			"hello\nworld",
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 4},
					End:   lsp.Position{Line: 1, Character: 0},
				},
				RangeLength: 2,
				Text:        " ",
			},
			"hell world",
		},
		"replace after astral": {
			`𐆓 beep 𐆓`,
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 4},
					End:   lsp.Position{Line: 0, Character: 6},
				},
				RangeLength: 2,
				Text:        "oo",
			},
			"𐆓 boop 𐆓",
		},
		"replace with astral": {
			`𐆓 beep 𐆓`,
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 4},
					End:   lsp.Position{Line: 0, Character: 6},
				},
				RangeLength: 2,
				Text:        "𐆓",
			},
			"𐆓 b𐆓p 𐆓",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			input := []byte(test.Input)
			want := []byte(test.Want)

			f := newFile("test")
			f.change(input)
			f.applyChange(test.Change)
			got := f.content
			if !bytes.Equal(got, want) {
				t.Errorf("wrong result\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}
