// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/providers"
)

func TestTfvarsencode(t *testing.T) {
	tableTestFunction(t, "encode_tfvars", []functionTest{
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal("hello"),
				"number": cty.NumberIntVal(5),
				"bool":   cty.True,
				"set":    cty.SetVal([]cty.Value{cty.StringVal("beep"), cty.StringVal("boop")}),
				"list":   cty.SetVal([]cty.Value{cty.StringVal("bleep"), cty.StringVal("bloop")}),
				"tuple":  cty.SetVal([]cty.Value{cty.StringVal("bibble"), cty.StringVal("wibble")}),
				"map":    cty.MapVal(map[string]cty.Value{"one": cty.NumberIntVal(1)}),
				"object": cty.ObjectVal(map[string]cty.Value{"one": cty.NumberIntVal(1), "true": cty.True}),
				"null":   cty.NullVal(cty.String),
			}),
			Want: cty.StringVal(
				`bool = true
list = ["bleep", "bloop"]
map = {
  one = 1
}
null   = null
number = 5
object = {
  one  = 1
  true = true
}
set    = ["beep", "boop"]
string = "hello"
tuple  = ["bibble", "wibble"]
`),
		},
		{
			Input: cty.EmptyObjectVal,
			Want:  cty.StringVal(``),
		},
		{
			Input: cty.MapVal(map[string]cty.Value{
				"one":   cty.NumberIntVal(1),
				"two":   cty.NumberIntVal(2),
				"three": cty.NumberIntVal(3),
			}),
			Want: cty.StringVal(
				`one   = 1
three = 3
two   = 2
`),
		},
		{
			Input: cty.MapValEmpty(cty.String),
			Want:  cty.StringVal(``),
		},
		{
			Input: cty.UnknownVal(cty.EmptyObject),
			Want:  cty.UnknownVal(cty.String).RefineNotNull(),
		},
		{
			Input: cty.UnknownVal(cty.Map(cty.String)),
			Want:  cty.UnknownVal(cty.String).RefineNotNull(),
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"string": cty.UnknownVal(cty.String),
			}),
			Want: cty.UnknownVal(cty.String).RefineNotNull(),
		},
		{
			Input: cty.MapVal(map[string]cty.Value{
				"string": cty.UnknownVal(cty.String),
			}),
			Want: cty.UnknownVal(cty.String).RefineNotNull(),
		},
		{
			Input:   cty.NullVal(cty.EmptyObject),
			WantErr: `cannot encode a null value in tfvars syntax`,
		},
		{
			Input:   cty.NullVal(cty.Map(cty.String)),
			WantErr: `cannot encode a null value in tfvars syntax`,
		},
		{
			Input:   cty.StringVal("nope"),
			WantErr: `invalid value to encode: must be an object whose attribute names will become the encoded variable names`,
		},
		{
			Input:   cty.Zero,
			WantErr: `invalid value to encode: must be an object whose attribute names will become the encoded variable names`,
		},
		{
			Input:   cty.False,
			WantErr: `invalid value to encode: must be an object whose attribute names will become the encoded variable names`,
		},
		{
			Input:   cty.ListValEmpty(cty.String),
			WantErr: `invalid value to encode: must be an object whose attribute names will become the encoded variable names`,
		},
		{
			Input:   cty.SetValEmpty(cty.String),
			WantErr: `invalid value to encode: must be an object whose attribute names will become the encoded variable names`,
		},
		{
			Input:   cty.EmptyTupleVal,
			WantErr: `invalid value to encode: must be an object whose attribute names will become the encoded variable names`,
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"not valid identifier": cty.StringVal("!"),
			}),
			WantErr: `invalid variable name "not valid identifier": must be a valid identifier, per Terraform's rules for input variable declarations`,
		},
	})
}

func TestTfvarsdecode(t *testing.T) {
	tableTestFunction(t, "decode_tfvars", []functionTest{
		{
			Input: cty.StringVal(`string = "hello"
number = 2`),
			Want: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal("hello"),
				"number": cty.NumberIntVal(2),
			}),
		},
		{
			Input: cty.StringVal(``),
			Want:  cty.EmptyObjectVal,
		},
		{
			Input: cty.UnknownVal(cty.String),
			Want:  cty.UnknownVal(cty.DynamicPseudoType),
		},
		{
			Input:   cty.NullVal(cty.String),
			WantErr: `cannot decode tfvars from a null value`,
		},
		{
			Input: cty.StringVal(`not valid syntax`),
			// This is actually not a very good diagnosis for this error,
			// since we're expecting HCL arguments rather than HCL blocks,
			// but that's something we'd need to address in HCL.
			WantErr: `invalid tfvars syntax: <decode_tfvars argument>:1,17-17: Invalid block definition; Either a quoted string block label or an opening brace ("{") is expected here.`,
		},
		{
			Input:   cty.StringVal(`foo = not valid syntax`),
			WantErr: `invalid tfvars syntax: <decode_tfvars argument>:1,11-16: Missing newline after argument; An argument definition must end with a newline.`,
		},
		{
			Input:   cty.StringVal(`foo = var.whatever`),
			WantErr: `invalid expression for variable "foo": <decode_tfvars argument>:1,7-10: Variables not allowed; Variables may not be used here.`,
		},
		{
			Input:   cty.StringVal(`foo = whatever()`),
			WantErr: `invalid expression for variable "foo": <decode_tfvars argument>:1,7-17: Function calls not allowed; Functions may not be called here.`,
		},
	})
}

func TestExprencode(t *testing.T) {
	tableTestFunction(t, "encode_expr", []functionTest{
		{
			Input: cty.StringVal("hello"),
			Want:  cty.StringVal(`"hello"`),
		},
		{
			Input: cty.StringVal("hello\nworld\n"),
			Want:  cty.StringVal(`"hello\nworld\n"`),
			// NOTE: If HCL changes the above to be a heredoc in future (which
			// would make this test fail) then our function's refinement
			// that unknown strings encode with the prefix " will become
			// invalid, and should be removed.
		},
		{
			Input: cty.StringVal("hel${lo"),
			Want:  cty.StringVal(`"hel$${lo"`), // Escape template interpolation sequence
		},
		{
			Input: cty.StringVal("hel%{lo"),
			Want:  cty.StringVal(`"hel%%{lo"`), // Escape template control sequence
		},
		{
			Input: cty.StringVal(`boop\boop`),
			Want:  cty.StringVal(`"boop\\boop"`), // Escape literal backslash
		},
		{
			Input: cty.StringVal(""),
			Want:  cty.StringVal(`""`),
		},
		{
			Input: cty.NumberIntVal(2),
			Want:  cty.StringVal(`2`),
		},
		{
			Input: cty.True,
			Want:  cty.StringVal(`true`),
		},
		{
			Input: cty.False,
			Want:  cty.StringVal(`false`),
		},
		{
			Input: cty.EmptyObjectVal,
			Want:  cty.StringVal(`{}`),
		},
		{
			Input: cty.ObjectVal(map[string]cty.Value{
				"number": cty.NumberIntVal(5),
				"string": cty.StringVal("..."),
			}),
			Want: cty.StringVal(`{
  number = 5
  string = "..."
}`),
		},
		{
			Input: cty.MapVal(map[string]cty.Value{
				"one": cty.NumberIntVal(1),
				"two": cty.NumberIntVal(2),
			}),
			Want: cty.StringVal(`{
  one = 1
  two = 2
}`),
		},
		{
			Input: cty.EmptyTupleVal,
			Want:  cty.StringVal(`[]`),
		},
		{
			Input: cty.TupleVal([]cty.Value{
				cty.NumberIntVal(5),
				cty.StringVal("..."),
			}),
			Want: cty.StringVal(`[5, "..."]`),
		},
		{
			Input: cty.SetVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(5),
				cty.NumberIntVal(20),
				cty.NumberIntVal(55),
			}),
			Want: cty.StringVal(`[1, 5, 20, 55]`),
		},
		{
			Input: cty.DynamicVal,
			Want:  cty.UnknownVal(cty.String).RefineNotNull(),
		},
		{
			Input: cty.UnknownVal(cty.Number).RefineNotNull(),
			Want:  cty.UnknownVal(cty.String).RefineNotNull(),
		},
		{
			Input: cty.UnknownVal(cty.String).RefineNotNull(),
			Want: cty.UnknownVal(cty.String).Refine().
				NotNull().
				StringPrefixFull(`"`).
				NewValue(),
		},
		{
			Input: cty.UnknownVal(cty.EmptyObject).RefineNotNull(),
			Want: cty.UnknownVal(cty.String).Refine().
				NotNull().
				StringPrefixFull(`{`).
				NewValue(),
		},
		{
			Input: cty.UnknownVal(cty.Map(cty.String)).RefineNotNull(),
			Want: cty.UnknownVal(cty.String).Refine().
				NotNull().
				StringPrefixFull(`{`).
				NewValue(),
		},
		{
			Input: cty.UnknownVal(cty.EmptyTuple).RefineNotNull(),
			Want: cty.UnknownVal(cty.String).Refine().
				NotNull().
				StringPrefixFull(`[`).
				NewValue(),
		},
		{
			Input: cty.UnknownVal(cty.List(cty.String)).RefineNotNull(),
			Want: cty.UnknownVal(cty.String).Refine().
				NotNull().
				StringPrefixFull(`[`).
				NewValue(),
		},
		{
			Input: cty.UnknownVal(cty.Set(cty.String)).RefineNotNull(),
			Want: cty.UnknownVal(cty.String).Refine().
				NotNull().
				StringPrefixFull(`[`).
				NewValue(),
		},
	})
}

type functionTest struct {
	Input   cty.Value
	Want    cty.Value
	WantErr string
}

func tableTestFunction(t *testing.T, functionName string, tests []functionTest) {
	t.Helper()

	provider := NewProvider()
	for _, test := range tests {
		t.Run(test.Input.GoString(), func(t *testing.T) {
			resp := provider.CallFunction(providers.CallFunctionRequest{
				FunctionName: functionName,
				Arguments:    []cty.Value{test.Input},
			})
			if test.WantErr != "" {
				err := resp.Err
				if err == nil {
					t.Fatalf("unexpected success for %#v; want error\ngot: %#v", test.Input, resp.Result)
				}
				if err.Error() != test.WantErr {
					t.Errorf("wrong error\ngot:  %s\nwant: %s", err.Error(), test.WantErr)
				}
				return
			}
			if resp.Err != nil {
				t.Fatalf("unexpected error: %s", resp.Err)
			}
			if diff := cmp.Diff(test.Want, resp.Result, ctydebug.CmpOptions); diff != "" {
				t.Errorf("wrong result for %#v\n%s", test.Input, diff)
			}
		})
	}
}
