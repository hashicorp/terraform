// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package format

import (
	"bytes"
	"fmt"
	"regexp"
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
		"error origination from failed test assertion": {
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Test assertion failed",
				Detail:   "LHS not equal to RHS",
				Subject: &hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 6, Byte: 5},
					End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
				},
				Expression: &hclsyntax.BinaryOpExpr{
					Op: hclsyntax.OpEqual,
					LHS: &hclsyntax.LiteralValueExpr{
						Val: cty.ObjectVal(map[string]cty.Value{
							"inner": cty.StringVal("str1"),
							"extra": cty.StringVal("str2"),
						}),
					},
					RHS: &hclsyntax.LiteralValueExpr{
						Val: cty.ObjectVal(map[string]cty.Value{
							"inner": cty.StringVal("str11"),
							"extra": cty.StringVal("str21"),
						}),
					},
					SrcRange: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 1, Column: 6, Byte: 5},
						End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
					},
				},
				EvalContext: &hcl.EvalContext{
					Variables: map[string]cty.Value{
						"foo": cty.ObjectVal(map[string]cty.Value{
							"inner": cty.StringVal("str1"),
						}),
						"bar": cty.ObjectVal(map[string]cty.Value{
							"inner": cty.StringVal("str2"),
						}),
					},
				},
				// This is simulating what the test assertion expression
				// type would generate on evaluation, by implementing the
				// same interface it uses.
				Extra: diagnosticCausedByTestFailure{true},
			},
			`[red]╷[reset]
[red]│[reset] [bold][red]Error: [reset][bold]Test assertion failed[reset]
[red]│[reset]
[red]│[reset]   on test.tf line 1:
[red]│[reset]    1: test [underline]source[reset] code
[red]│[reset]     [dark_gray]├────────────────[reset]
[red]│[reset]     [dark_gray]│[reset] [bold]LHS[reset]:
[red]│[reset]     [dark_gray]│[reset]   {
[red]│[reset]     [dark_gray]│[reset]     "extra": "str2",
[red]│[reset]     [dark_gray]│[reset]     "inner": "str1"
[red]│[reset]     [dark_gray]│[reset]   }
[red]│[reset]     [dark_gray]│[reset] [bold]RHS[reset]:
[red]│[reset]     [dark_gray]│[reset]   {
[red]│[reset]     [dark_gray]│[reset]     "extra": "str21",
[red]│[reset]     [dark_gray]│[reset]     "inner": "str11"
[red]│[reset]     [dark_gray]│[reset]   }
[red]│[reset]     [dark_gray]│[reset] [bold]Diff[reset]:
[red]│[reset]     [dark_gray]│[reset] [red][bold]--- actual[reset]
[red]│[reset]     [dark_gray]│[reset] [green][bold]+++ expected[reset]
[red]│[reset]     [dark_gray]│[reset]  [reset] {
[red]│[reset]     [dark_gray]│[reset] [red]-[reset]   "extra": "str2",
[red]│[reset]     [dark_gray]│[reset] [red]-[reset]   "inner": "str1"
[red]│[reset]     [dark_gray]│[reset] [green]+[reset]   "extra": "str21",
[red]│[reset]     [dark_gray]│[reset] [green]+[reset]   "inner": "str11"
[red]│[reset]     [dark_gray]│[reset]  [reset] }
[red]│[reset]
[red]│[reset]
[red]│[reset] LHS not equal to RHS
[red]╵[reset]
`,
		},
		"error originating from failed wrapped test assertion by function": {
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Test assertion failed",
				Detail:   "Example crash",
				Subject: &hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 6, Byte: 5},
					End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
				},
				Expression: &hclsyntax.FunctionCallExpr{
					Name: "tobool",
					Args: []hclsyntax.Expression{
						&hclsyntax.BinaryOpExpr{
							Op: hclsyntax.OpEqual,
							LHS: &hclsyntax.LiteralValueExpr{
								Val: cty.ObjectVal(map[string]cty.Value{
									"inner": cty.StringVal("str1"),
									"extra": cty.StringVal("str2"),
								}),
							},
							RHS: &hclsyntax.LiteralValueExpr{
								Val: cty.ObjectVal(map[string]cty.Value{
									"inner": cty.StringVal("str11"),
									"extra": cty.StringVal("str21"),
								}),
							},
							SrcRange: hcl.Range{
								Filename: "test.tf",
								Start:    hcl.Pos{Line: 1, Column: 6, Byte: 5},
								End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
							},
						},
					},
				},
				EvalContext: &hcl.EvalContext{
					Variables: map[string]cty.Value{},
					Functions: map[string]function.Function{
						"tobool": function.New(&function.Spec{
							Params: []function.Parameter{
								{
									Name: "param_0",
									Type: cty.String,
								},
							},
						}),
					},
				},
				// This is simulating what the test assertion expression
				// type would generate on evaluation, by implementing the
				// same interface it uses.
				Extra: diagnosticCausedByTestFailure{true},
			},
			`[red]╷[reset]
[red]│[reset] [bold][red]Error: [reset][bold]Test assertion failed[reset]
[red]│[reset]
[red]│[reset]   on test.tf line 1:
[red]│[reset]    1: test [underline]source[reset] code
[red]│[reset]
[red]│[reset] Example crash
[red]╵[reset]
`,
		},
		"warning from deprecation": {
			&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Deprecation detected",
				Detail:   "Countermeasures must be taken.",
				Subject: &hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 6, Byte: 5},
					End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
				},
				Extra: &tfdiags.DeprecationOriginDiagnosticExtra{
					OriginDescription: "module.foo.bar",
				},
			},
			`[yellow]╷[reset]
[yellow]│[reset] [bold][yellow]Warning: [reset][bold]Deprecation detected[reset]
[yellow]│[reset]
[yellow]│[reset]   on test.tf line 1:
[yellow]│[reset]    1: test [underline]source[reset] code
[yellow]│[reset]
[yellow]│[reset]   The deprecation originates from module.foo.bar
[yellow]│[reset]
[yellow]│[reset] Countermeasures must be taken.
[yellow]╵[reset]
`,
		},
	}

	sources := map[string][]byte{
		"test.tf":       []byte(`test source code`),
		"deprecated.tf": []byte(`source of deprecation`),
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

			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("wrong result\ngot:\n%s\n\nwant:\n%s\n\ndiff:\n%s\n\n", got, want, diff)
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

		"warning from deprecation": {
			&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Deprecation detected",
				Detail:   "Countermeasures must be taken.",
				Subject: &hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 6, Byte: 5},
					End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
				},
				Extra: &tfdiags.DeprecationOriginDiagnosticExtra{
					OriginDescription: "module.foo.bar",
				},
			},
			`
Warning: Deprecation detected

  on test.tf line 1:
   1: test source code

  The deprecation originates from module.foo.bar

Countermeasures must be taken.
`,
		},
	}

	sources := map[string][]byte{
		"test.tf":       []byte(`test source code`),
		"deprecated.tf": []byte(`source of deprecation`),
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var diags tfdiags.Diagnostics
			diags = diags.Append(test.Diag) // to normalize it into a tfdiag.Diagnostic
			diag := diags[0]
			got := strings.TrimSpace(DiagnosticPlain(diag, sources, 40))
			want := strings.TrimSpace(test.Want)
			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("wrong result\ngot:\n%s\n\nwant:\n%s\n\n,diff:\n%s\n\n", got, want, diff)
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

func TestDiagnosticPlainFromJSON_policySnippet(t *testing.T) {
	tests := map[string]struct {
		diag *viewsjson.Diagnostic
		want string
	}{
		"with policy range, policy snippet, and range": {
			diag: &viewsjson.Diagnostic{
				Severity: viewsjson.DiagnosticSeverityError,
				Summary:  "policy denied",
				PolicyRange: &viewsjson.DiagnosticRange{
					Filename: "policy_file.tfpolicy.hcl",
					Start:    viewsjson.Pos{Line: 1},
				},
				PolicySnippet: &viewsjson.DiagnosticSnippet{
					Code:                 `resource_policy "resource_type" "policy_name" {`,
					StartLine:            1,
					HighlightStartOffset: 0,
					HighlightEndOffset:   48,
				},
				Range: &viewsjson.DiagnosticRange{
					Filename: "main.tf",
					Start:    viewsjson.Pos{Line: 1},
				},
				Snippet: &viewsjson.DiagnosticSnippet{
					Code:                 `resource "test_instance" "foo" {`,
					StartLine:            1,
					HighlightStartOffset: 0,
					HighlightEndOffset:   30,
				},
			},
			want: `
Error: policy denied

  on policy_file.tfpolicy.hcl line 1:
   1: resource_policy "resource_type" "policy_name" {

  while evaluating policy for main.tf line 1:
   1: resource "test_instance" "foo" {
`,
		},
		"with policy range and policy snippet but no range": {
			diag: &viewsjson.Diagnostic{
				Severity: viewsjson.DiagnosticSeverityError,
				Summary:  "policy denied",
				PolicyRange: &viewsjson.DiagnosticRange{
					Filename: "policy_file.tfpolicy.hcl",
					Start:    viewsjson.Pos{Line: 1},
				},
				PolicySnippet: &viewsjson.DiagnosticSnippet{
					Code:                 `resource_policy "resource_type" "policy_name" {`,
					StartLine:            1,
					HighlightStartOffset: 0,
					HighlightEndOffset:   48,
				},
			},
			want: `
Error: policy denied

  on policy_file.tfpolicy.hcl line 1:
   1: resource_policy "resource_type" "policy_name" {
`,
		},
		"with policy values of failing members": {
			diag: policyMembersDiag(2),
			want: `
Error: Condition not met

  on policies/files.policy.hcl line 3, in resource_policy "local_file" "permissions":
   3:     condition = every(original, attrs.file_permission <= each.attrs.file_permission)
    ├────────────────
    │ attrs.file_permission is "0600"
    │ original.attrs.file_permission is "0644" (local_file.readme)
    │ original.attrs.file_permission is "0640" (local_file.notes)
    │ original.attrs.file_permission is "0444" (local_file.legal)
    │ (and 2 more)

Files must not be more permissive than the original.
`,
		},
		"with policy values of failing members, none omitted": {
			diag: policyMembersDiag(0),
			want: `
Error: Condition not met

  on policies/files.policy.hcl line 3, in resource_policy "local_file" "permissions":
   3:     condition = every(original, attrs.file_permission <= each.attrs.file_permission)
    ├────────────────
    │ attrs.file_permission is "0600"
    │ original.attrs.file_permission is "0644" (local_file.readme)
    │ original.attrs.file_permission is "0640" (local_file.notes)
    │ original.attrs.file_permission is "0444" (local_file.legal)

Files must not be more permissive than the original.
`,
		},
		// Policy values keep the policy engine's order, while the values of
		// the configuration snippet are sorted.
		"with policy values in the engine's order and sorted configuration values": {
			diag: &viewsjson.Diagnostic{
				Severity: viewsjson.DiagnosticSeverityError,
				Summary:  "Condition not met",
				PolicyRange: &viewsjson.DiagnosticRange{
					Filename: "policies/files.policy.hcl",
					Start:    viewsjson.Pos{Line: 3},
				},
				PolicySnippet: &viewsjson.DiagnosticSnippet{
					Code:      `    condition = every(original, each.attrs.owner == attrs.owner)`,
					StartLine: 3,
					Values: []viewsjson.DiagnosticExpressionValue{
						{Traversal: "attrs.owner", Statement: `is "alice"`},
						{Traversal: "original.attrs.owner", Statement: `is "bob" (local_file.readme)`},
						{Traversal: "original.attrs.mode", Statement: `is "0644" (local_file.readme)`},
						{Traversal: "original.attrs.owner", Statement: `is "carol" (local_file.notes)`},
						{Traversal: "original.attrs.mode", Statement: `is "0640" (local_file.notes)`},
					},
				},
				Range: &viewsjson.DiagnosticRange{
					Filename: "main.tf",
					Start:    viewsjson.Pos{Line: 1},
				},
				Snippet: &viewsjson.DiagnosticSnippet{
					Code:      `  owner = var.owner`,
					StartLine: 1,
					Values: []viewsjson.DiagnosticExpressionValue{
						{Traversal: "var.owner", Statement: `is "alice"`},
						{Traversal: "var.mode", Statement: `is "0600"`},
					},
				},
			},
			want: `
Error: Condition not met

  on policies/files.policy.hcl line 3:
   3:     condition = every(original, each.attrs.owner == attrs.owner)
    ├────────────────
    │ attrs.owner is "alice"
    │ original.attrs.owner is "bob" (local_file.readme)
    │ original.attrs.mode is "0644" (local_file.readme)
    │ original.attrs.owner is "carol" (local_file.notes)
    │ original.attrs.mode is "0640" (local_file.notes)

  while evaluating policy for main.tf line 1:
   1:   owner = var.owner
    ├────────────────
    │ var.mode is "0600"
    │ var.owner is "alice"
`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := strings.TrimSpace(DiagnosticPlainFromJSON(tc.diag, 0))
			want := strings.TrimSpace(tc.want)

			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("unexpected output diff:\n%s", diff)
			}
		})
	}
}

// The line about omitted failing members has the style of the value lines,
// without a traversal.
func TestDiagnosticFromJSON_policyOmittedMembers(t *testing.T) {
	// This empty Colorize just passes through all of the formatting codes
	// untouched, because it doesn't define any formatting keywords.
	got := DiagnosticFromJSON(policyMembersDiag(2), &colorstring.Colorize{}, 0)
	want := `[red]│[reset]     [dark_gray]│[reset] [bold]original.attrs.file_permission[reset] is "0444" (local_file.legal)
[red]│[reset]     [dark_gray]│[reset] (and 2 more)
`
	if !strings.Contains(got, want) {
		t.Fatalf("missing omitted members line\ngot:\n%s\nwant it to contain:\n%s", got, want)
	}
}

// policyMembersDiag returns a policy diagnostic of a failed every condition
// with the values of the subject and of three failing members.
func policyMembersDiag(omittedMembers int) *viewsjson.Diagnostic {
	context := `resource_policy "local_file" "permissions"`
	return &viewsjson.Diagnostic{
		Severity: viewsjson.DiagnosticSeverityError,
		Summary:  "Condition not met",
		Detail:   "Files must not be more permissive than the original.",
		PolicyRange: &viewsjson.DiagnosticRange{
			Filename: "policies/files.policy.hcl",
			Start:    viewsjson.Pos{Line: 3, Column: 17, Byte: 70},
			End:      viewsjson.Pos{Line: 3, Column: 85, Byte: 138},
		},
		PolicySnippet: &viewsjson.DiagnosticSnippet{
			Context:              &context,
			Code:                 `    condition = every(original, attrs.file_permission <= each.attrs.file_permission)`,
			StartLine:            3,
			HighlightStartOffset: 16,
			HighlightEndOffset:   84,
			Values: []viewsjson.DiagnosticExpressionValue{
				{Traversal: "attrs.file_permission", Statement: `is "0600"`},
				{Traversal: "original.attrs.file_permission", Statement: `is "0644" (local_file.readme)`},
				{Traversal: "original.attrs.file_permission", Statement: `is "0640" (local_file.notes)`},
				{Traversal: "original.attrs.file_permission", Statement: `is "0444" (local_file.legal)`},
			},
			OmittedMembers: omittedMembers,
		},
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

func TestJsonDiff(t *testing.T) {
	f := &snippetFormatter{
		buf: &bytes.Buffer{},
		color: &colorstring.Colorize{
			Reset:   true,
			Disable: true,
		},
	}

	tests := []struct {
		name string
		strA string
		strB string
		diff string
	}{
		{
			name: "Basic different fields",
			strA: `{
  "field1": "value1",
  "field2": "value2",
  "field3": "value3",
  "field4": "value4"
}`,
			strB: `{
  "field1": "value1",
  "field2": "different",
  "field3": "value3",
  "field4": "value4"
}`,
			diff: `    │ Diff:
    │ --- actual
    │ +++ expected
    │   {
    │     "field1": "value1",
    │ -   "field2": "value2",
    │ +   "field2": "different",
    │     "field3": "value3",
    │     "field4": "value4"
    │   }
`,
		},
		{
			name: "Unequal number of fields",
			strA: `{
  "field1": "value1",
  "field2": "value2",
  "field3": "value3",
  "extraField": "extraValue",
  "field4": "value4"
}`,
			strB: `{
  "field1": "value1",
  "fieldX": "valueX",
  "fieldY": "valueY",
  "field4": "value4"
}`,
			diff: `    │ Diff:
    │ --- actual
    │ +++ expected
    │   {
    │     "field1": "value1",
    │ -   "field2": "value2",
    │ -   "field3": "value3",
    │ -   "extraField": "extraValue",
    │ -   "field4": "value4"
    │ - }
    │ +   "fieldX": "valueX",
    │ +   "fieldY": "valueY",
    │ +   "field4": "value4"
    │ + }
`,
		},
		{
			name: "Empty vs non-empty JSON",
			strA: `{}`,
			strB: `{
  "field1": "value1",
  "field2": "value2"
}`,
			diff: `    │ Diff:
    │ --- actual
    │ +++ expected
    │ - {}
    │ + {
    │ +   "field1": "value1",
    │ +   "field2": "value2"
    │ + }
`,
		},
		{
			name: "Completely different JSONs",
			strA: `{
  "a": 1,
  "b": 2
}`,
			strB: `{
  "c": 3,
  "d": 4
}`,
			diff: `    │ Diff:
    │ --- actual
    │ +++ expected
    │   {
    │ -   "a": 1,
    │ -   "b": 2
    │ +   "c": 3,
    │ +   "d": 4
    │   }
`,
		},
		{
			name: "Nested objects with differences",
			strA: `{
  "outer": {
    "inner1": "value1",
    "inner2": "value2"
  }
}`,
			strB: `{
  "outer": {
    "inner1": "changed",
    "inner2": "value2"
  }
}`,
			diff: `    │ Diff:
    │ --- actual
    │ +++ expected
    │   {
    │     "outer": {
    │ -     "inner1": "value1",
    │ +     "inner1": "changed",
    │       "inner2": "value2"
    │     }
    │   }
`,
		},
		{
			name: "Multiple separate diff blocks",
			strA: `{
  "block1": "original1",
  "unchanged1": "same",
  "block2": "original2",
  "unchanged2": "same",
  "block3": "original3"
}`,
			strB: `{
  "block1": "changed1",
  "unchanged1": "same",
  "block2": "changed2",
  "unchanged2": "same",
  "block3": "changed3"
}`,
			diff: `    │ Diff:
    │ --- actual
    │ +++ expected
    │   {
    │ -   "block1": "original1",
    │ +   "block1": "changed1",
    │     "unchanged1": "same",
    │ -   "block2": "original2",
    │ +   "block2": "changed2",
    │     "unchanged2": "same",
    │ -   "block3": "original3"
    │ +   "block3": "changed3"
    │   }
`,
		},
		{
			name: "Large number of differences",
			strA: `{
  "item1": "a",
  "item2": "b",
  "item3": "c",
  "item4": "d",
  "item5": "e",
  "item6": "f",
  "item7": "g"
}`,
			strB: `{
  "item1": "a",
  "item2": "B",
  "item3": "C",
  "item4": "D",
  "item5": "e",
  "item6": "F",
  "item7": "g"
}`,
			diff: `    │ Diff:
    │ --- actual
    │ +++ expected
    │   {
    │     "item1": "a",
    │ -   "item2": "b",
    │ -   "item3": "c",
    │ -   "item4": "d",
    │ +   "item2": "B",
    │ +   "item3": "C",
    │ +   "item4": "D",
    │     "item5": "e",
    │ -   "item6": "f",
    │ +   "item6": "F",
    │     "item7": "g"
    │   }
`,
		},
		{
			name: "Identical JSONs",
			strA: `{"field": "value"}`,
			strB: `{"field": "value"}`,
			diff: ``, // No output expected for identical JSONs
		},
		{
			name: "simple: no matches",
			strA: `1`,
			strB: `2`,
			diff: `    │ Diff:
    │ --- actual
    │ +++ expected
    │ - 1
    │ + 2
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f.buf.Reset()
			f.printJSONDiff(test.strA, test.strB)
			diff := regexp.MustCompile(`\[[^\]]+\]`).ReplaceAllString(f.buf.String(), "")
			fmt.Println(diff)
			if d := cmp.Diff(diff, test.diff); d != "" {
				t.Errorf("diff mismatch: got %s\n, want %s\n: diff: %s\n", diff, test.diff, d)
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

var _ tfdiags.DiagnosticExtraCausedByTestFailure = diagnosticCausedByTestFailure{}

type diagnosticCausedByTestFailure struct {
	Verbose bool
}

func (e diagnosticCausedByTestFailure) DiagnosticCausedByTestFailure() bool {
	return true
}

func (e diagnosticCausedByTestFailure) IsTestVerboseMode() bool {
	return e.Verbose
}
