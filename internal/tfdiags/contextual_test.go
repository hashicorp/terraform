// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfdiags

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func TestAttributeValue(t *testing.T) {
	testConfig := `
foo {
  bar = "hi"
}
foo {
  bar = "bar"
}
bar {
  bar = "woot"
}
baz "a" {
  bar = "beep"
}
baz "b" {
  bar = "boop"
}
parent {
  nested_str = "hello"
  nested_str_tuple = ["aa", "bbb", "cccc"]
  nested_num_tuple = [1, 9863, 22]
  nested_map = {
    first_key  = "first_value"
    second_key = "2nd value"
  }
}
tuple_of_one = ["one"]
tuple_of_two = ["first", "22222"]
root_map = {
  first  = "1st"
  second = "2nd"
}
simple_attr = "val"
`
	// TODO: Test ConditionalExpr
	// TODO: Test ForExpr
	// TODO: Test FunctionCallExpr
	// TODO: Test IndexExpr
	// TODO: Test interpolation
	// TODO: Test SplatExpr

	f, parseDiags := hclsyntax.ParseConfig([]byte(testConfig), "test.tf", hcl.Pos{Line: 1, Column: 1})
	if len(parseDiags) != 0 {
		t.Fatal(parseDiags)
	}
	emptySrcRng := &SourceRange{
		Filename: "test.tf",
		Start:    SourcePos{Line: 1, Column: 1, Byte: 0},
		End:      SourcePos{Line: 1, Column: 1, Byte: 0},
	}

	testCases := []struct {
		Diag          Diagnostic
		ExpectedRange *SourceRange
	}{
		{
			AttributeValue(
				Error,
				"foo[0].bar",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "foo"},
					cty.IndexStep{Key: cty.NumberIntVal(0)},
					cty.GetAttrStep{Name: "bar"},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 3, Column: 9, Byte: 15},
				End:      SourcePos{Line: 3, Column: 13, Byte: 19},
			},
		},
		{
			AttributeValue(
				Error,
				"foo[1].bar",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "foo"},
					cty.IndexStep{Key: cty.NumberIntVal(1)},
					cty.GetAttrStep{Name: "bar"},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 6, Column: 9, Byte: 36},
				End:      SourcePos{Line: 6, Column: 14, Byte: 41},
			},
		},
		{
			AttributeValue(
				Error,
				"foo[99].bar",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "foo"},
					cty.IndexStep{Key: cty.NumberIntVal(99)},
					cty.GetAttrStep{Name: "bar"},
				},
			),
			emptySrcRng,
		},
		{
			AttributeValue(
				Error,
				"bar.bar",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "bar"},
					cty.GetAttrStep{Name: "bar"},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 9, Column: 9, Byte: 58},
				End:      SourcePos{Line: 9, Column: 15, Byte: 64},
			},
		},
		{
			AttributeValue(
				Error,
				`baz["a"].bar`,
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "baz"},
					cty.IndexStep{Key: cty.StringVal("a")},
					cty.GetAttrStep{Name: "bar"},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 12, Column: 9, Byte: 85},
				End:      SourcePos{Line: 12, Column: 15, Byte: 91},
			},
		},
		{
			AttributeValue(
				Error,
				`baz["b"].bar`,
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "baz"},
					cty.IndexStep{Key: cty.StringVal("b")},
					cty.GetAttrStep{Name: "bar"},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 15, Column: 9, Byte: 112},
				End:      SourcePos{Line: 15, Column: 15, Byte: 118},
			},
		},
		{
			AttributeValue(
				Error,
				`baz["not_exists"].bar`,
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "baz"},
					cty.IndexStep{Key: cty.StringVal("not_exists")},
					cty.GetAttrStep{Name: "bar"},
				},
			),
			emptySrcRng,
		},
		{
			// Attribute value with subject already populated should not be disturbed.
			// (in a real case, this might've been passed through from a deeper function
			// in the call stack, for example.)
			&attributeDiagnostic{
				attrPath: cty.Path{cty.GetAttrStep{Name: "foo"}},
				diagnosticBase: diagnosticBase{
					summary: "preexisting",
					detail:  "detail",
					address: "original",
				},
				subject: &SourceRange{
					Filename: "somewhere_else.tf",
				},
			},
			&SourceRange{
				Filename: "somewhere_else.tf",
			},
		},
		{
			// Missing path
			&attributeDiagnostic{
				diagnosticBase: diagnosticBase{
					summary: "missing path",
				},
			},
			nil,
		},

		// Nested attributes
		{
			AttributeValue(
				Error,
				"parent.nested_str",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "parent"},
					cty.GetAttrStep{Name: "nested_str"},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 18, Column: 16, Byte: 145},
				End:      SourcePos{Line: 18, Column: 23, Byte: 152},
			},
		},
		{
			AttributeValue(
				Error,
				"parent.nested_str_tuple[99]",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "parent"},
					cty.GetAttrStep{Name: "nested_str_tuple"},
					cty.IndexStep{Key: cty.NumberIntVal(99)},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 19, Column: 3, Byte: 155},
				End:      SourcePos{Line: 19, Column: 19, Byte: 171},
			},
		},
		{
			AttributeValue(
				Error,
				"parent.nested_str_tuple[0]",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "parent"},
					cty.GetAttrStep{Name: "nested_str_tuple"},
					cty.IndexStep{Key: cty.NumberIntVal(0)},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 19, Column: 23, Byte: 175},
				End:      SourcePos{Line: 19, Column: 27, Byte: 179},
			},
		},
		{
			AttributeValue(
				Error,
				"parent.nested_str_tuple[2]",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "parent"},
					cty.GetAttrStep{Name: "nested_str_tuple"},
					cty.IndexStep{Key: cty.NumberIntVal(2)},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 19, Column: 36, Byte: 188},
				End:      SourcePos{Line: 19, Column: 42, Byte: 194},
			},
		},
		{
			AttributeValue(
				Error,
				"parent.nested_num_tuple[0]",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "parent"},
					cty.GetAttrStep{Name: "nested_num_tuple"},
					cty.IndexStep{Key: cty.NumberIntVal(0)},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 20, Column: 23, Byte: 218},
				End:      SourcePos{Line: 20, Column: 24, Byte: 219},
			},
		},
		{
			AttributeValue(
				Error,
				"parent.nested_num_tuple[1]",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "parent"},
					cty.GetAttrStep{Name: "nested_num_tuple"},
					cty.IndexStep{Key: cty.NumberIntVal(1)},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 20, Column: 26, Byte: 221},
				End:      SourcePos{Line: 20, Column: 30, Byte: 225},
			},
		},
		{
			AttributeValue(
				Error,
				"parent.nested_map.first_key",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "parent"},
					cty.GetAttrStep{Name: "nested_map"},
					cty.IndexStep{Key: cty.StringVal("first_key")},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 22, Column: 19, Byte: 266},
				End:      SourcePos{Line: 22, Column: 30, Byte: 277},
			},
		},
		{
			AttributeValue(
				Error,
				"parent.nested_map.second_key",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "parent"},
					cty.GetAttrStep{Name: "nested_map"},
					cty.IndexStep{Key: cty.StringVal("second_key")},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 23, Column: 19, Byte: 297},
				End:      SourcePos{Line: 23, Column: 28, Byte: 306},
			},
		},
		{
			AttributeValue(
				Error,
				"parent.nested_map.undefined_key",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "parent"},
					cty.GetAttrStep{Name: "nested_map"},
					cty.IndexStep{Key: cty.StringVal("undefined_key")},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 21, Column: 3, Byte: 233},
				End:      SourcePos{Line: 21, Column: 13, Byte: 243},
			},
		},

		// Root attributes of complex types
		{
			AttributeValue(
				Error,
				"tuple_of_one[0]",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "tuple_of_one"},
					cty.IndexStep{Key: cty.NumberIntVal(0)},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 26, Column: 17, Byte: 330},
				End:      SourcePos{Line: 26, Column: 22, Byte: 335},
			},
		},
		{
			AttributeValue(
				Error,
				"tuple_of_two[0]",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "tuple_of_two"},
					cty.IndexStep{Key: cty.NumberIntVal(0)},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 27, Column: 17, Byte: 353},
				End:      SourcePos{Line: 27, Column: 24, Byte: 360},
			},
		},
		{
			AttributeValue(
				Error,
				"tuple_of_two[1]",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "tuple_of_two"},
					cty.IndexStep{Key: cty.NumberIntVal(1)},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 27, Column: 26, Byte: 362},
				End:      SourcePos{Line: 27, Column: 33, Byte: 369},
			},
		},
		{
			AttributeValue(
				Error,
				"tuple_of_one[null]",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "tuple_of_one"},
					cty.IndexStep{Key: cty.NullVal(cty.Number)},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 26, Column: 1, Byte: 314},
				End:      SourcePos{Line: 26, Column: 13, Byte: 326},
			},
		},
		{
			// index out of range
			AttributeValue(
				Error,
				"tuple_of_two[99]",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "tuple_of_two"},
					cty.IndexStep{Key: cty.NumberIntVal(99)},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 27, Column: 1, Byte: 337},
				End:      SourcePos{Line: 27, Column: 13, Byte: 349},
			},
		},
		{
			AttributeValue(
				Error,
				"root_map.first",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "root_map"},
					cty.IndexStep{Key: cty.StringVal("first")},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 29, Column: 13, Byte: 396},
				End:      SourcePos{Line: 29, Column: 16, Byte: 399},
			},
		},
		{
			AttributeValue(
				Error,
				"root_map.second",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "root_map"},
					cty.IndexStep{Key: cty.StringVal("second")},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 30, Column: 13, Byte: 413},
				End:      SourcePos{Line: 30, Column: 16, Byte: 416},
			},
		},
		{
			AttributeValue(
				Error,
				"root_map.undefined_key",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "root_map"},
					cty.IndexStep{Key: cty.StringVal("undefined_key")},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 28, Column: 1, Byte: 371},
				End:      SourcePos{Line: 28, Column: 9, Byte: 379},
			},
		},
		{
			AttributeValue(
				Error,
				"simple_attr",
				"detail",
				cty.Path{
					cty.GetAttrStep{Name: "simple_attr"},
				},
			),
			&SourceRange{
				Filename: "test.tf",
				Start:    SourcePos{Line: 32, Column: 15, Byte: 434},
				End:      SourcePos{Line: 32, Column: 20, Byte: 439},
			},
		},
		{
			// This should never happen as error should always point to an attribute
			// or index of an attribute, but we should not crash if it does
			AttributeValue(
				Error,
				"key",
				"index_step",
				cty.Path{
					cty.IndexStep{Key: cty.StringVal("key")},
				},
			),
			emptySrcRng,
		},
		{
			// This should never happen as error should always point to an attribute
			// or index of an attribute, but we should not crash if it does
			AttributeValue(
				Error,
				"key.another",
				"index_step",
				cty.Path{
					cty.IndexStep{Key: cty.StringVal("key")},
					cty.IndexStep{Key: cty.StringVal("another")},
				},
			),
			emptySrcRng,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d:%s", i, tc.Diag.Description()), func(t *testing.T) {
			var diags Diagnostics

			origAddr := tc.Diag.Description().Address
			diags = diags.Append(tc.Diag)

			gotDiags := diags.InConfigBody(f.Body, "test.addr")
			gotRange := gotDiags[0].Source().Subject
			gotAddr := gotDiags[0].Description().Address

			switch {
			case origAddr != "":
				if gotAddr != origAddr {
					t.Errorf("original diagnostic address modified from %s to %s", origAddr, gotAddr)
				}
			case gotAddr != "test.addr":
				t.Error("missing detail address")
			}

			for _, problem := range deep.Equal(gotRange, tc.ExpectedRange) {
				t.Error(problem)
			}
		})
	}
}

func TestGetAttribute(t *testing.T) {
	path := cty.Path{
		cty.GetAttrStep{Name: "foo"},
		cty.IndexStep{Key: cty.NumberIntVal(0)},
		cty.GetAttrStep{Name: "bar"},
	}

	d := AttributeValue(
		Error,
		"foo[0].bar",
		"detail",
		path,
	)

	p := GetAttribute(d)
	if !reflect.DeepEqual(path, p) {
		t.Fatalf("paths don't match:\nexpected: %#v\ngot: %#v", path, p)
	}
}
