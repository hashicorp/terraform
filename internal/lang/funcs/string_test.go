// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/customdecode"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/hashicorp/terraform/internal/collections"
)

func TestReplace(t *testing.T) {
	tests := []struct {
		String  cty.Value
		Substr  cty.Value
		Replace cty.Value
		Want    cty.Value
		Err     bool
	}{
		{ // Regular search and replace
			cty.StringVal("hello"),
			cty.StringVal("hel"),
			cty.StringVal("bel"),
			cty.StringVal("bello"),
			false,
		},
		{ // Search string doesn't match
			cty.StringVal("hello"),
			cty.StringVal("nope"),
			cty.StringVal("bel"),
			cty.StringVal("hello"),
			false,
		},
		{ // Regular expression
			cty.StringVal("hello"),
			cty.StringVal("/l/"),
			cty.StringVal("L"),
			cty.StringVal("heLLo"),
			false,
		},
		{
			cty.StringVal("helo"),
			cty.StringVal("/(l)/"),
			cty.StringVal("$1$1"),
			cty.StringVal("hello"),
			false,
		},
		{ // Bad regexp
			cty.StringVal("hello"),
			cty.StringVal("/(l/"),
			cty.StringVal("$1$1"),
			cty.UnknownVal(cty.String),
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("replace(%#v, %#v, %#v)", test.String, test.Substr, test.Replace), func(t *testing.T) {
			got, err := Replace(test.String, test.Substr, test.Replace)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestStrContains(t *testing.T) {
	tests := []struct {
		String cty.Value
		Substr cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("hello"),
			cty.StringVal("hel"),
			cty.BoolVal(true),
			false,
		},
		{
			cty.StringVal("hello"),
			cty.StringVal("lo"),
			cty.BoolVal(true),
			false,
		},
		{
			cty.StringVal("hello1"),
			cty.StringVal("1"),
			cty.BoolVal(true),
			false,
		},
		{
			cty.StringVal("hello1"),
			cty.StringVal("heo"),
			cty.BoolVal(false),
			false,
		},
		{
			cty.StringVal("hello1"),
			cty.NumberIntVal(1),
			cty.UnknownVal(cty.Bool),
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("includes(%#v, %#v)", test.String, test.Substr), func(t *testing.T) {
			got, err := StrContains(test.String, test.Substr)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestStartsWith(t *testing.T) {
	tests := []struct {
		String, Prefix cty.Value
		Want           cty.Value
		WantError      string
	}{
		{
			cty.StringVal("hello world"),
			cty.StringVal("hello"),
			cty.True,
			``,
		},
		{
			cty.StringVal("hey world"),
			cty.StringVal("hello"),
			cty.False,
			``,
		},
		{
			cty.StringVal(""),
			cty.StringVal(""),
			cty.True,
			``,
		},
		{
			cty.StringVal("a"),
			cty.StringVal(""),
			cty.True,
			``,
		},
		{
			cty.StringVal(""),
			cty.StringVal("a"),
			cty.False,
			``,
		},
		{
			cty.UnknownVal(cty.String),
			cty.StringVal("a"),
			cty.UnknownVal(cty.Bool).RefineNotNull(),
			``,
		},
		{
			cty.UnknownVal(cty.String),
			cty.StringVal(""),
			cty.True,
			``,
		},
		{
			cty.UnknownVal(cty.String).Refine().StringPrefix("https:").NewValue(),
			cty.StringVal(""),
			cty.True,
			``,
		},
		{
			cty.UnknownVal(cty.String).Refine().StringPrefix("https:").NewValue(),
			cty.StringVal("a"),
			cty.False,
			``,
		},
		{
			cty.UnknownVal(cty.String).Refine().StringPrefix("https:").NewValue(),
			cty.StringVal("ht"),
			cty.True,
			``,
		},
		{
			cty.UnknownVal(cty.String).Refine().StringPrefix("https:").NewValue(),
			cty.StringVal("https:"),
			cty.True,
			``,
		},
		{
			cty.UnknownVal(cty.String).Refine().StringPrefix("https:").NewValue(),
			cty.StringVal("https-"),
			cty.False,
			``,
		},
		{
			cty.UnknownVal(cty.String).Refine().StringPrefix("https:").NewValue(),
			cty.StringVal("https://"),
			cty.UnknownVal(cty.Bool).RefineNotNull(),
			``,
		},
		{
			// Unicode combining characters edge-case: we match the prefix
			// in terms of unicode code units rather than grapheme clusters,
			// which is inconsistent with our string processing elsewhere but
			// would be a breaking change to fix that bug now.
			cty.StringVal("\U0001f937\u200d\u2642"), // "Man Shrugging" is encoded as "Person Shrugging" followed by zero-width joiner and then the masculine gender presentation modifier
			cty.StringVal("\U0001f937"),             // Just the "Person Shrugging" character without any modifiers
			cty.True,
			``,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("StartsWith(%#v, %#v)", test.String, test.Prefix), func(t *testing.T) {
			got, err := StartsWithFunc.Call([]cty.Value{test.String, test.Prefix})

			if test.WantError != "" {
				gotErr := fmt.Sprintf("%s", err)
				if gotErr != test.WantError {
					t.Errorf("wrong error\ngot:  %s\nwant: %s", gotErr, test.WantError)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf(
					"wrong result\nstring: %#v\nprefix: %#v\ngot:    %#v\nwant:   %#v",
					test.String, test.Prefix, got, test.Want,
				)
			}
		})
	}
}

func TestTemplateString(t *testing.T) {
	// This function has some special restrictions on what syntax is valid
	// in its first argument, so we'll test this one using HCL expressions
	// as the inputs, rather than direct cty values as we do for most other
	// functions in this package.
	tests := []struct {
		templateExpr string
		exprScope    map[string]cty.Value
		vars         cty.Value
		want         cty.Value
		wantErr      string
	}{
		{
			`template`,
			map[string]cty.Value{
				"template": cty.StringVal(`it's ${a}`),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
			}),
			cty.StringVal(`it's a value`),
			``,
		},
		{
			`template`,
			map[string]cty.Value{
				"template": cty.StringVal(`${a}`),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.True,
			}),
			// The special treatment of a template with only a single
			// interpolation sequence does not apply to templatestring, because
			// we're expecting to be evaluating templates fetched dynamically
			// from somewhere else and want to avoid callers needing to deal
			// with anything other than string results.
			cty.StringVal(`true`),
			``,
		},
		{
			`template`,
			map[string]cty.Value{
				"template": cty.StringVal(`${a}`),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.EmptyTupleVal,
			}),
			// The special treatment of a template with only a single
			// interpolation sequence does not apply to templatestring, because
			// we're expecting to be evaluating templates fetched dynamically
			// from somewhere else and want to avoid callers needing to deal
			// with anything other than string results.
			cty.NilVal,
			`invalid template result: string required`,
		},
		{
			`data.whatever.whatever["foo"].result`,
			map[string]cty.Value{
				"data": cty.ObjectVal(map[string]cty.Value{
					"whatever": cty.ObjectVal(map[string]cty.Value{
						"whatever": cty.MapVal(map[string]cty.Value{
							"foo": cty.ObjectVal(map[string]cty.Value{
								"result": cty.StringVal("it's ${a}"),
							}),
						}),
					}),
				}),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
			}),
			cty.StringVal(`it's a value`),
			``,
		},
		{
			`data.whatever.whatever[each.key].result`,
			map[string]cty.Value{
				"data": cty.ObjectVal(map[string]cty.Value{
					"whatever": cty.ObjectVal(map[string]cty.Value{
						"whatever": cty.MapVal(map[string]cty.Value{
							"foo": cty.ObjectVal(map[string]cty.Value{
								"result": cty.StringVal("it's ${a}"),
							}),
						}),
					}),
				}),
				"each": cty.ObjectVal(map[string]cty.Value{
					"key": cty.StringVal("foo"),
				}),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
			}),
			cty.StringVal(`it's a value`),
			``,
		},
		{
			`data.whatever.whatever[*].result`,
			map[string]cty.Value{
				"data": cty.ObjectVal(map[string]cty.Value{
					"whatever": cty.ObjectVal(map[string]cty.Value{
						"whatever": cty.TupleVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"result": cty.StringVal("it's ${a}"),
							}),
						}),
					}),
				}),
				"each": cty.ObjectVal(map[string]cty.Value{
					"key": cty.StringVal("foo"),
				}),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
			}),
			cty.NilVal,
			// We have an intentional hole in our heuristic for whether the
			// first argument is a suitable expression which permits splat
			// expressions just so that we can return the type mismatch error
			// from the result not being a string, instead of the more general
			// error about it not being a supported expression type.
			`invalid template value: a string is required`,
		},
		{
			`"can't write $${not_allowed}"`,
			map[string]cty.Value{},
			cty.ObjectVal(map[string]cty.Value{
				"not_allowed": cty.StringVal("a literal template"),
			}),
			cty.NilVal,
			`invalid template expression: templatestring is only for rendering templates retrieved dynamically from elsewhere, and so does not support providing a literal template; consider using a template string expression instead`,
		},
		{
			`"can't write ${not_allowed}"`,
			map[string]cty.Value{},
			cty.ObjectVal(map[string]cty.Value{
				"not_allowed": cty.StringVal("a literal template"),
			}),
			cty.NilVal,
			`invalid template expression: templatestring is only for rendering templates retrieved dynamically from elsewhere; to render an inline template, consider using a plain template string expression`,
		},
		{
			`"can't write %%{for x in things}a literal template%%{endfor}"`,
			map[string]cty.Value{},
			cty.ObjectVal(map[string]cty.Value{
				"things": cty.ListVal([]cty.Value{cty.True}),
			}),
			cty.NilVal,
			`invalid template expression: templatestring is only for rendering templates retrieved dynamically from elsewhere, and so does not support providing a literal template; consider using a template string expression instead`,
		},
		{
			`"can't write %{for x in things}a literal template%{endfor}"`,
			map[string]cty.Value{},
			cty.ObjectVal(map[string]cty.Value{
				"things": cty.ListVal([]cty.Value{cty.True}),
			}),
			cty.NilVal,
			`invalid template expression: templatestring is only for rendering templates retrieved dynamically from elsewhere; to render an inline template, consider using a plain template string expression`,
		},
		{
			`"${not_allowed}"`,
			map[string]cty.Value{},
			cty.ObjectVal(map[string]cty.Value{
				"not allowed": cty.StringVal("an interp-only template"),
			}),
			cty.NilVal,
			`invalid template expression: templatestring is only for rendering templates retrieved dynamically from elsewhere; to treat the inner expression as template syntax, write the reference expression directly without any template interpolation syntax`,
		},
		{
			`1 + 1`,
			map[string]cty.Value{},
			cty.ObjectVal(map[string]cty.Value{}),
			cty.NilVal,
			`invalid template expression: must be a direct reference to a single string from elsewhere, containing valid Terraform template syntax`,
		},
		{
			`not_a_string`,
			map[string]cty.Value{
				"not_a_string": cty.True,
			},
			cty.ObjectVal(map[string]cty.Value{}),
			cty.NilVal,
			`invalid template value: a string is required`,
		},
		{
			`with_lower`,
			map[string]cty.Value{
				"with_lower": cty.StringVal(`it's ${lower(a)}`),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("A VALUE"),
			}),
			cty.StringVal("it's a value"),
			``,
		},
		{
			`with_core_lower`,
			map[string]cty.Value{
				"with_core_lower": cty.StringVal(`it's ${core::lower(a)}`),
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("A VALUE"),
			}),
			cty.StringVal("it's a value"),
			``,
		},
		{
			`with_fsfunc`,
			map[string]cty.Value{
				"with_fsfunc": cty.StringVal(`it's ${fsfunc()}`),
			},
			cty.ObjectVal(map[string]cty.Value{}),
			cty.NilVal,
			`<templatestring argument>:1,8-15: Error in function call; Call to function "fsfunc" failed: cannot use filesystem access functions like fsfunc in templatestring templates; consider passing the function result as a template variable instead.`,
		},
		{
			`with_core_fsfunc`,
			map[string]cty.Value{
				"with_core_fsfunc": cty.StringVal(`it's ${core::fsfunc()}`),
			},
			cty.ObjectVal(map[string]cty.Value{}),
			cty.NilVal,
			`<templatestring argument>:1,8-21: Error in function call; Call to function "core::fsfunc" failed: cannot use filesystem access functions like fsfunc in templatestring templates; consider passing the function result as a template variable instead.`,
		},
		{
			`with_templatefunc`,
			map[string]cty.Value{
				"with_templatefunc": cty.StringVal(`it's ${templatefunc()}`),
			},
			cty.ObjectVal(map[string]cty.Value{}),
			cty.NilVal,
			`<templatestring argument>:1,8-21: Error in function call; Call to function "templatefunc" failed: cannot recursively call templatefunc from inside another template function.`,
		},
		{
			`with_core_templatefunc`,
			map[string]cty.Value{
				"with_core_templatefunc": cty.StringVal(`it's ${core::templatefunc()}`),
			},
			cty.ObjectVal(map[string]cty.Value{}),
			cty.NilVal,
			`<templatestring argument>:1,8-27: Error in function call; Call to function "core::templatefunc" failed: cannot recursively call templatefunc from inside another template function.`,
		},
		{
			`with_fstemplatefunc`,
			map[string]cty.Value{
				"with_fstemplatefunc": cty.StringVal(`it's ${fstemplatefunc()}`),
			},
			cty.ObjectVal(map[string]cty.Value{}),
			cty.NilVal,
			// The template function error takes priority over the filesystem
			// function error if calling a function that's in both categories.
			`<templatestring argument>:1,8-23: Error in function call; Call to function "fstemplatefunc" failed: cannot recursively call fstemplatefunc from inside another template function.`,
		},
		{
			`with_core_fstemplatefunc`,
			map[string]cty.Value{
				"with_core_fstemplatefunc": cty.StringVal(`it's ${core::fstemplatefunc()}`),
			},
			cty.ObjectVal(map[string]cty.Value{}),
			cty.NilVal,
			// The template function error takes priority over the filesystem
			// function error if calling a function that's in both categories.
			`<templatestring argument>:1,8-29: Error in function call; Call to function "core::fstemplatefunc" failed: cannot recursively call fstemplatefunc from inside another template function.`,
		},
	}

	funcToTest := MakeTemplateStringFunc(func() (funcs map[string]function.Function, fsFuncs collections.Set[string], templateFuncs collections.Set[string]) {
		// These are the functions available for use inside the nested template
		// evaluation context. These are here only to test that we can call
		// functions and that the template/filesystem functions get blocked
		// with suitable error messages. This is not a realistic set of
		// functions that would be available in a real call.
		funcs = map[string]function.Function{
			"lower": function.New(&function.Spec{
				Params: []function.Parameter{
					{
						Name: "str",
						Type: cty.String,
					},
				},
				Type: function.StaticReturnType(cty.String),
				Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
					s := args[0].AsString()
					return cty.StringVal(strings.ToLower(s)), nil
				},
			}),
			"fsfunc": function.New(&function.Spec{
				Type: function.StaticReturnType(cty.String),
				Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
					return cty.UnknownVal(retType), fmt.Errorf("should not be able to call fsfunc")
				},
			}),
			"templatefunc": function.New(&function.Spec{
				Type: function.StaticReturnType(cty.String),
				Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
					return cty.UnknownVal(retType), fmt.Errorf("should not be able to call templatefunc")
				},
			}),
			"fstemplatefunc": function.New(&function.Spec{
				Type: function.StaticReturnType(cty.String),
				Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
					return cty.UnknownVal(retType), fmt.Errorf("should not be able to call fstemplatefunc")
				},
			}),
		}
		funcs["core::lower"] = funcs["lower"]
		funcs["core::fsfunc"] = funcs["fsfunc"]
		funcs["core::templatefunc"] = funcs["templatefunc"]
		funcs["core::fstemplatefunc"] = funcs["fstemplatefunc"]
		return funcs, collections.NewSetCmp("fsfunc", "fstemplatefunc"), collections.NewSetCmp("templatefunc", "fstemplatefunc")
	})

	for _, test := range tests {
		t.Run(test.templateExpr, func(t *testing.T) {
			// The following mimics what HCL itself would do when preparing
			// the first argument to this function, since the parameter
			// uses the special "expression closure type" which causes
			// HCL to delay evaluation of the expression and let the
			// function handle it directly itself.
			expr, diags := hclsyntax.ParseExpression([]byte(test.templateExpr), "", hcl.InitialPos)
			if diags.HasErrors() {
				t.Fatalf("unexpected errors: %s", diags.Error())
			}
			exprClosure := &customdecode.ExpressionClosure{
				Expression: expr,
				EvalContext: &hcl.EvalContext{
					Variables: test.exprScope,
				},
			}
			exprClosureVal := customdecode.ExpressionClosureVal(exprClosure)

			got, gotErr := funcToTest.Call([]cty.Value{exprClosureVal, test.vars})

			if test.wantErr != "" {
				if gotErr == nil {
					t.Fatalf("unexpected success\ngot: %#v\nwant error: %s", got, test.wantErr)
				}
				if got, want := gotErr.Error(), test.wantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			}
			if gotErr != nil {
				t.Errorf("unexpected error: %s", gotErr.Error())
			}
			if !test.want.RawEquals(got) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}
