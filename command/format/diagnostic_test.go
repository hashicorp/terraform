package format

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/mitchellh/colorstring"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/tfdiags"
)

func TestDiagnostic(t *testing.T) {

	tests := map[string]struct {
		Diag interface{}
		Want string
	}{
		"sourceless error": {
			tfdiags.Sourceless(
				tfdiags.Error,
				"A sourceless error",
				"It has no source references but it does have a pretty long detail that should wrap over multiple lines.",
			),
			`
[bold][red]Error: [reset][bold]A sourceless error[reset]

It has no source references but it does
have a pretty long detail that should
wrap over multiple lines.
`,
		},
		"sourceless warning": {
			tfdiags.Sourceless(
				tfdiags.Warning,
				"A sourceless warning",
				"It has no source references but it does have a pretty long detail that should wrap over multiple lines.",
			),
			`
[bold][yellow]Warning: [reset][bold]A sourceless warning[reset]

It has no source references but it does
have a pretty long detail that should
wrap over multiple lines.
`,
		},
		"error with source code subject": {
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Bad bad bad",
				Detail:   "Whatever shall we do?",
				Subject: &hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 6, Byte: 5},
					End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
				},
			},
			`
[bold][red]Error: [reset][bold]Bad bad bad[reset]

  on test.tf line 1:
   1: test [underline]source[reset] code

Whatever shall we do?
`,
		},
		"error with source code subject and known expression": {
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Bad bad bad",
				Detail:   "Whatever shall we do?",
				Subject: &hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 6, Byte: 5},
					End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
				},
				Expression: hcltest.MockExprTraversal(hcl.Traversal{
					hcl.TraverseRoot{Name: "boop"},
					hcl.TraverseAttr{Name: "beep"},
				}),
				EvalContext: &hcl.EvalContext{
					Variables: map[string]cty.Value{
						"boop": cty.ObjectVal(map[string]cty.Value{
							"beep": cty.StringVal("blah"),
						}),
					},
				},
			},
			`
[bold][red]Error: [reset][bold]Bad bad bad[reset]

  on test.tf line 1:
   1: test [underline]source[reset] code
    [dark_gray]|----------------[reset]
    [dark_gray]|[reset] [bold]boop.beep[reset] is "blah"

Whatever shall we do?
`,
		},
		"error with source code subject and expression referring to sensitive value": {
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Bad bad bad",
				Detail:   "Whatever shall we do?",
				Subject: &hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 6, Byte: 5},
					End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
				},
				Expression: hcltest.MockExprTraversal(hcl.Traversal{
					hcl.TraverseRoot{Name: "boop"},
					hcl.TraverseAttr{Name: "beep"},
				}),
				EvalContext: &hcl.EvalContext{
					Variables: map[string]cty.Value{
						"boop": cty.ObjectVal(map[string]cty.Value{
							"beep": cty.StringVal("blah").Mark("sensitive"),
						}),
					},
				},
			},
			`
[bold][red]Error: [reset][bold]Bad bad bad[reset]

  on test.tf line 1:
   1: test [underline]source[reset] code
    [dark_gray]|----------------[reset]
    [dark_gray]|[reset] [bold]boop.beep[reset] has a sensitive value

Whatever shall we do?
`,
		},
		"error with source code subject and unknown string expression": {
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Bad bad bad",
				Detail:   "Whatever shall we do?",
				Subject: &hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 6, Byte: 5},
					End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
				},
				Expression: hcltest.MockExprTraversal(hcl.Traversal{
					hcl.TraverseRoot{Name: "boop"},
					hcl.TraverseAttr{Name: "beep"},
				}),
				EvalContext: &hcl.EvalContext{
					Variables: map[string]cty.Value{
						"boop": cty.ObjectVal(map[string]cty.Value{
							"beep": cty.UnknownVal(cty.String),
						}),
					},
				},
			},
			`
[bold][red]Error: [reset][bold]Bad bad bad[reset]

  on test.tf line 1:
   1: test [underline]source[reset] code
    [dark_gray]|----------------[reset]
    [dark_gray]|[reset] [bold]boop.beep[reset] is a string, known only after apply

Whatever shall we do?
`,
		},
		"error with source code subject and unknown expression of unknown type": {
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Bad bad bad",
				Detail:   "Whatever shall we do?",
				Subject: &hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 6, Byte: 5},
					End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
				},
				Expression: hcltest.MockExprTraversal(hcl.Traversal{
					hcl.TraverseRoot{Name: "boop"},
					hcl.TraverseAttr{Name: "beep"},
				}),
				EvalContext: &hcl.EvalContext{
					Variables: map[string]cty.Value{
						"boop": cty.ObjectVal(map[string]cty.Value{
							"beep": cty.UnknownVal(cty.DynamicPseudoType),
						}),
					},
				},
			},
			`
[bold][red]Error: [reset][bold]Bad bad bad[reset]

  on test.tf line 1:
   1: test [underline]source[reset] code
    [dark_gray]|----------------[reset]
    [dark_gray]|[reset] [bold]boop.beep[reset] will be known only after apply

Whatever shall we do?
`,
		},
	}

	sources := map[string][]byte{
		"test.tf": []byte(`test source code`),
	}

	// This empty Colorize just passes through all of the formatting codes
	// untouched, because it doesn't define any formatting keywords.
	colorize := &colorstring.Colorize{}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var diags tfdiags.Diagnostics
			diags = diags.Append(test.Diag) // to normalize it into a tfdiag.Diagnostic
			diag := diags[0]
			got := strings.TrimSpace(Diagnostic(diag, sources, colorize, 40))
			want := strings.TrimSpace(test.Want)
			if got != want {
				t.Errorf("wrong result\ngot:\n%s\n\nwant:\n%s\n\n", got, want)
			}
		})
	}
}

func TestDiagnosticWarningsCompact(t *testing.T) {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.SimpleWarning("foo"))
	diags = diags.Append(tfdiags.SimpleWarning("foo"))
	diags = diags.Append(tfdiags.SimpleWarning("bar"))
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "source foo",
		Detail:   "...",
		Subject: &hcl.Range{
			Filename: "source.tf",
			Start:    hcl.Pos{Line: 2, Column: 1, Byte: 5},
			End:      hcl.Pos{Line: 2, Column: 1, Byte: 5},
		},
	})
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "source foo",
		Detail:   "...",
		Subject: &hcl.Range{
			Filename: "source.tf",
			Start:    hcl.Pos{Line: 3, Column: 1, Byte: 7},
			End:      hcl.Pos{Line: 3, Column: 1, Byte: 7},
		},
	})
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "source bar",
		Detail:   "...",
		Subject: &hcl.Range{
			Filename: "source2.tf",
			Start:    hcl.Pos{Line: 1, Column: 1, Byte: 1},
			End:      hcl.Pos{Line: 1, Column: 1, Byte: 1},
		},
	})

	// ConsolidateWarnings groups together the ones
	// that have source location information and that
	// have the same summary text.
	diags = diags.ConsolidateWarnings(1)

	// A zero-value Colorize just passes all the formatting
	// codes back to us, so we can test them literally.
	got := DiagnosticWarningsCompact(diags, &colorstring.Colorize{})
	want := `[bold][yellow]Warnings:[reset]

- foo
- foo
- bar
- source foo
  on source.tf line 2 (and 1 more)
- source bar
  on source2.tf line 1
`
	if got != want {
		t.Errorf(
			"wrong result\ngot:\n%s\n\nwant:\n%s\n\ndiff:\n%s",
			got, want, cmp.Diff(want, got),
		)
	}
}

// Test case via https://github.com/hashicorp/terraform/issues/21359
func TestDiagnostic_nonOverlappingHighlightContext(t *testing.T) {
	var diags tfdiags.Diagnostics

	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Some error",
		Detail:   "...",
		Subject: &hcl.Range{
			Filename: "source.tf",
			Start:    hcl.Pos{Line: 1, Column: 5, Byte: 5},
			End:      hcl.Pos{Line: 1, Column: 5, Byte: 5},
		},
		Context: &hcl.Range{
			Filename: "source.tf",
			Start:    hcl.Pos{Line: 1, Column: 5, Byte: 5},
			End:      hcl.Pos{Line: 4, Column: 2, Byte: 60},
		},
	})
	sources := map[string][]byte{
		"source.tf": []byte(`x = somefunc("testing", {
  alpha = "foo"
  beta  = "bar"
})
`),
	}
	color := &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Reset:   true,
		Disable: true,
	}
	expected := `
Error: Some error

  on source.tf line 1:
   1: x = somefunc("testing", {
   2:   alpha = "foo"
   3:   beta  = "bar"
   4: })

...
`
	output := Diagnostic(diags[0], sources, color, 80)

	if output != expected {
		t.Fatalf("unexpected output: got:\n%s\nwant\n%s\n", output, expected)
	}
}

func TestDiagnostic_emptyOverlapHighlightContext(t *testing.T) {
	var diags tfdiags.Diagnostics

	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Some error",
		Detail:   "...",
		Subject: &hcl.Range{
			Filename: "source.tf",
			Start:    hcl.Pos{Line: 3, Column: 10, Byte: 38},
			End:      hcl.Pos{Line: 4, Column: 1, Byte: 39},
		},
		Context: &hcl.Range{
			Filename: "source.tf",
			Start:    hcl.Pos{Line: 2, Column: 13, Byte: 27},
			End:      hcl.Pos{Line: 4, Column: 1, Byte: 39},
		},
	})
	sources := map[string][]byte{
		"source.tf": []byte(`variable "x" {
  default = {
    "foo"
  }
`),
	}
	color := &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Reset:   true,
		Disable: true,
	}
	expected := `
Error: Some error

  on source.tf line 3, in variable "x":
   2:   default = {
   3:     "foo"
   4:   }

...
`
	output := Diagnostic(diags[0], sources, color, 80)

	if output != expected {
		t.Fatalf("unexpected output: got:\n%s\nwant\n%s\n", output, expected)
	}
}

func TestDiagnostic_wrapDetailIncludingCommand(t *testing.T) {
	var diags tfdiags.Diagnostics

	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Everything went wrong",
		Detail:   "This is a very long sentence about whatever went wrong which is supposed to wrap onto multiple lines. Thank-you very much for listening.\n\nTo fix this, run this very long command:\n  terraform read-my-mind -please -thanks -but-do-not-wrap-this-line-because-it-is-prefixed-with-spaces\n\nHere is a coda which is also long enough to wrap and so it should eventually make it onto multiple lines. THE END",
	})
	color := &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Reset:   true,
		Disable: true,
	}
	expected := `
Error: Everything went wrong

This is a very long sentence about whatever went wrong which is supposed to
wrap onto multiple lines. Thank-you very much for listening.

To fix this, run this very long command:
  terraform read-my-mind -please -thanks -but-do-not-wrap-this-line-because-it-is-prefixed-with-spaces

Here is a coda which is also long enough to wrap and so it should eventually
make it onto multiple lines. THE END
`
	output := Diagnostic(diags[0], nil, color, 76)

	if output != expected {
		t.Fatalf("unexpected output: got:\n%s\nwant\n%s\n", output, expected)
	}
}
