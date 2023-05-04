// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package format

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/mitchellh/colorstring"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/lang/marks"

	"github.com/hashicorp/terraform/internal/tfdiags"
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
			`[red]╷[reset]
[red]│[reset] [bold][red]Error: [reset][bold]A sourceless error[reset]
[red]│[reset]
[red]│[reset] It has no source references but it
[red]│[reset] does have a pretty long detail that
[red]│[reset] should wrap over multiple lines.
[red]╵[reset]
`,
		},
		"sourceless warning": {
			tfdiags.Sourceless(
				tfdiags.Warning,
				"A sourceless warning",
				"It has no source references but it does have a pretty long detail that should wrap over multiple lines.",
			),
			`[yellow]╷[reset]
[yellow]│[reset] [bold][yellow]Warning: [reset][bold]A sourceless warning[reset]
[yellow]│[reset]
[yellow]│[reset] It has no source references but it
[yellow]│[reset] does have a pretty long detail that
[yellow]│[reset] should wrap over multiple lines.
[yellow]╵[reset]
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
			`[red]╷[reset]
[red]│[reset] [bold][red]Error: [reset][bold]Bad bad bad[reset]
[red]│[reset]
[red]│[reset]   on test.tf line 1:
[red]│[reset]    1: test [underline]source[reset] code
[red]│[reset]
[red]│[reset] Whatever shall we do?
[red]╵[reset]
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
			`[red]╷[reset]
[red]│[reset] [bold][red]Error: [reset][bold]Bad bad bad[reset]
[red]│[reset]
[red]│[reset]   on test.tf line 1:
[red]│[reset]    1: test [underline]source[reset] code
[red]│[reset]     [dark_gray]├────────────────[reset]
[red]│[reset]     [dark_gray]│[reset] [bold]boop.beep[reset] is "blah"
[red]│[reset]
[red]│[reset] Whatever shall we do?
[red]╵[reset]
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
							"beep": cty.StringVal("blah").Mark(marks.Sensitive),
						}),
					},
				},
				Extra: diagnosticCausedBySensitive(true),
			},
			`[red]╷[reset]
[red]│[reset] [bold][red]Error: [reset][bold]Bad bad bad[reset]
[red]│[reset]
[red]│[reset]   on test.tf line 1:
[red]│[reset]    1: test [underline]source[reset] code
[red]│[reset]     [dark_gray]├────────────────[reset]
[red]│[reset]     [dark_gray]│[reset] [bold]boop.beep[reset] has a sensitive value
[red]│[reset]
[red]│[reset] Whatever shall we do?
[red]╵[reset]
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
				Extra: diagnosticCausedByUnknown(true),
			},
			`[red]╷[reset]
[red]│[reset] [bold][red]Error: [reset][bold]Bad bad bad[reset]
[red]│[reset]
[red]│[reset]   on test.tf line 1:
[red]│[reset]    1: test [underline]source[reset] code
[red]│[reset]     [dark_gray]├────────────────[reset]
[red]│[reset]     [dark_gray]│[reset] [bold]boop.beep[reset] is a string, known only after apply
[red]│[reset]
[red]│[reset] Whatever shall we do?
[red]╵[reset]
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
				Extra: diagnosticCausedByUnknown(true),
			},
			`[red]╷[reset]
[red]│[reset] [bold][red]Error: [reset][bold]Bad bad bad[reset]
[red]│[reset]
[red]│[reset]   on test.tf line 1:
[red]│[reset]    1: test [underline]source[reset] code
[red]│[reset]     [dark_gray]├────────────────[reset]
[red]│[reset]     [dark_gray]│[reset] [bold]boop.beep[reset] will be known only after apply
[red]│[reset]
[red]│[reset] Whatever shall we do?
[red]╵[reset]
`,
		},
		"error with source code subject and function call annotation": {
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Bad bad bad",
				Detail:   "Whatever shall we do?",
				Subject: &hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 6, Byte: 5},
					End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
				},
				Expression: hcltest.MockExprLiteral(cty.True),
				EvalContext: &hcl.EvalContext{
					Functions: map[string]function.Function{
						"beep": function.New(&function.Spec{
							Params: []function.Parameter{
								{
									Name: "pos_param_0",
									Type: cty.String,
								},
								{
									Name: "pos_param_1",
									Type: cty.Number,
								},
							},
							VarParam: &function.Parameter{
								Name: "var_param",
								Type: cty.Bool,
							},
						}),
					},
				},
				// This is simulating what the HCL function call expression
				// type would generate on evaluation, by implementing the
				// same interface it uses.
				Extra: fakeDiagFunctionCallExtra("beep"),
			},
			`[red]╷[reset]
[red]│[reset] [bold][red]Error: [reset][bold]Bad bad bad[reset]
[red]│[reset]
[red]│[reset]   on test.tf line 1:
[red]│[reset]    1: test [underline]source[reset] code
[red]│[reset]     [dark_gray]├────────────────[reset]
[red]│[reset]     [dark_gray]│[reset] while calling [bold]beep[reset](pos_param_0, pos_param_1, var_param...)
[red]│[reset]
[red]│[reset] Whatever shall we do?
[red]╵[reset]
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

func TestDiagnosticPlain(t *testing.T) {

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
Error: A sourceless error

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
Warning: A sourceless warning

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
Error: Bad bad bad

  on test.tf line 1:
   1: test source code

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
Error: Bad bad bad

  on test.tf line 1:
   1: test source code
    ├────────────────
    │ boop.beep is "blah"

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
							"beep": cty.StringVal("blah").Mark(marks.Sensitive),
						}),
					},
				},
				Extra: diagnosticCausedBySensitive(true),
			},
			`
Error: Bad bad bad

  on test.tf line 1:
   1: test source code
    ├────────────────
    │ boop.beep has a sensitive value

Whatever shall we do?
`,
		},
		"error with source code subject and expression referring to sensitive value when not related to sensitivity": {
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
							"beep": cty.StringVal("blah").Mark(marks.Sensitive),
						}),
					},
				},
			},
			`
Error: Bad bad bad

  on test.tf line 1:
   1: test source code

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
				Extra: diagnosticCausedByUnknown(true),
			},
			`
Error: Bad bad bad

  on test.tf line 1:
   1: test source code
    ├────────────────
    │ boop.beep is a string, known only after apply

Whatever shall we do?
`,
		},
		"error with source code subject and unknown string expression when problem isn't unknown-related": {
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
Error: Bad bad bad

  on test.tf line 1:
   1: test source code
    ├────────────────
    │ boop.beep is a string

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
				Extra: diagnosticCausedByUnknown(true),
			},
			`
Error: Bad bad bad

  on test.tf line 1:
   1: test source code
    ├────────────────
    │ boop.beep will be known only after apply

Whatever shall we do?
`,
		},
		"error with source code subject and unknown expression of unknown type when problem isn't unknown-related": {
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
Error: Bad bad bad

  on test.tf line 1:
   1: test source code

Whatever shall we do?
`,
		},
	}

	sources := map[string][]byte{
		"test.tf": []byte(`test source code`),
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var diags tfdiags.Diagnostics
			diags = diags.Append(test.Diag) // to normalize it into a tfdiag.Diagnostic
			diag := diags[0]
			got := strings.TrimSpace(DiagnosticPlain(diag, sources, 40))
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
	expected := `╷
│ Error: Some error
│
│   on source.tf line 1:
│    1: x = somefunc("testing", {
│    2:   alpha = "foo"
│    3:   beta  = "bar"
│    4: })
│
│ ...
╵
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
	expected := `╷
│ Error: Some error
│
│   on source.tf line 3, in variable "x":
│    2:   default = {
│    3:     "foo"
│    4:   }
│
│ ...
╵
`
	output := Diagnostic(diags[0], sources, color, 80)

	if output != expected {
		t.Fatalf("unexpected output: got:\n%s\nwant\n%s\n", output, expected)
	}
}

func TestDiagnosticPlain_emptyOverlapHighlightContext(t *testing.T) {
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

	expected := `
Error: Some error

  on source.tf line 3, in variable "x":
   2:   default = {
   3:     "foo"
   4:   }

...
`
	output := DiagnosticPlain(diags[0], sources, 80)

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
	expected := `╷
│ Error: Everything went wrong
│
│ This is a very long sentence about whatever went wrong which is supposed
│ to wrap onto multiple lines. Thank-you very much for listening.
│
│ To fix this, run this very long command:
│   terraform read-my-mind -please -thanks -but-do-not-wrap-this-line-because-it-is-prefixed-with-spaces
│
│ Here is a coda which is also long enough to wrap and so it should
│ eventually make it onto multiple lines. THE END
╵
`
	output := Diagnostic(diags[0], nil, color, 76)

	if output != expected {
		t.Fatalf("unexpected output: got:\n%s\nwant\n%s\n", output, expected)
	}
}

func TestDiagnosticPlain_wrapDetailIncludingCommand(t *testing.T) {
	var diags tfdiags.Diagnostics

	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Everything went wrong",
		Detail:   "This is a very long sentence about whatever went wrong which is supposed to wrap onto multiple lines. Thank-you very much for listening.\n\nTo fix this, run this very long command:\n  terraform read-my-mind -please -thanks -but-do-not-wrap-this-line-because-it-is-prefixed-with-spaces\n\nHere is a coda which is also long enough to wrap and so it should eventually make it onto multiple lines. THE END",
	})

	expected := `
Error: Everything went wrong

This is a very long sentence about whatever went wrong which is supposed to
wrap onto multiple lines. Thank-you very much for listening.

To fix this, run this very long command:
  terraform read-my-mind -please -thanks -but-do-not-wrap-this-line-because-it-is-prefixed-with-spaces

Here is a coda which is also long enough to wrap and so it should
eventually make it onto multiple lines. THE END
`
	output := DiagnosticPlain(diags[0], nil, 76)

	if output != expected {
		t.Fatalf("unexpected output: got:\n%s\nwant\n%s\n", output, expected)
	}
}

// Test cases covering invalid JSON diagnostics which should still render
// correctly. These JSON diagnostic values cannot be generated from the
// json.NewDiagnostic code path, but we may read and display JSON diagnostics
// in future from other sources.
func TestDiagnosticFromJSON_invalid(t *testing.T) {
	tests := map[string]struct {
		Diag *viewsjson.Diagnostic
		Want string
	}{
		"zero-value end range and highlight end byte": {
			&viewsjson.Diagnostic{
				Severity: viewsjson.DiagnosticSeverityError,
				Summary:  "Bad end",
				Detail:   "It all went wrong.",
				Range: &viewsjson.DiagnosticRange{
					Filename: "ohno.tf",
					Start:    viewsjson.Pos{Line: 1, Column: 23, Byte: 22},
					End:      viewsjson.Pos{Line: 0, Column: 0, Byte: 0},
				},
				Snippet: &viewsjson.DiagnosticSnippet{
					Code:                 `resource "foo_bar "baz" {`,
					StartLine:            1,
					HighlightStartOffset: 22,
					HighlightEndOffset:   0,
				},
			},
			`[red]╷[reset]
[red]│[reset] [bold][red]Error: [reset][bold]Bad end[reset]
[red]│[reset]
[red]│[reset]   on ohno.tf line 1:
[red]│[reset]    1: resource "foo_bar "baz[underline]"[reset] {
[red]│[reset]
[red]│[reset] It all went wrong.
[red]╵[reset]
`,
		},
	}

	// This empty Colorize just passes through all of the formatting codes
	// untouched, because it doesn't define any formatting keywords.
	colorize := &colorstring.Colorize{}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := strings.TrimSpace(DiagnosticFromJSON(test.Diag, colorize, 40))
			want := strings.TrimSpace(test.Want)
			if got != want {
				t.Errorf("wrong result\ngot:\n%s\n\nwant:\n%s\n\n", got, want)
			}
		})
	}
}

// fakeDiagFunctionCallExtra is a fake implementation of the interface that
// HCL uses to provide "extra information" associated with diagnostics that
// describe errors during a function call.
type fakeDiagFunctionCallExtra string

var _ hclsyntax.FunctionCallDiagExtra = fakeDiagFunctionCallExtra("")

func (e fakeDiagFunctionCallExtra) CalledFunctionName() string {
	return string(e)
}

func (e fakeDiagFunctionCallExtra) FunctionCallError() error {
	return nil
}

// diagnosticCausedByUnknown is a testing helper for exercising our logic
// for selectively showing unknown values alongside our source snippets for
// diagnostics that are explicitly marked as being caused by unknown values.
type diagnosticCausedByUnknown bool

var _ tfdiags.DiagnosticExtraBecauseUnknown = diagnosticCausedByUnknown(true)

func (e diagnosticCausedByUnknown) DiagnosticCausedByUnknown() bool {
	return bool(e)
}

// diagnosticCausedBySensitive is a testing helper for exercising our logic
// for selectively showing sensitive values alongside our source snippets for
// diagnostics that are explicitly marked as being caused by sensitive values.
type diagnosticCausedBySensitive bool

var _ tfdiags.DiagnosticExtraBecauseSensitive = diagnosticCausedBySensitive(true)

func (e diagnosticCausedBySensitive) DiagnosticCausedBySensitive() bool {
	return bool(e)
}
