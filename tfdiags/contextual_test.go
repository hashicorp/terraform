package tfdiags

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
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
`
	f, parseDiags := hclsyntax.ParseConfig([]byte(testConfig), "test.tf", hcl.Pos{Line: 1, Column: 1})
	if len(parseDiags) != 0 {
		t.Fatal(parseDiags)
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
			// Attribute value with subject already populated should not be disturbed.
			// (in a real case, this might've been passed through from a deeper function
			// in the call stack, for example.)
			&attributeDiagnostic{
				diagnosticBase: diagnosticBase{
					summary: "preexisting",
					detail:  "detail",
				},
				subject: &SourceRange{
					Filename: "somewhere_else.tf",
				},
			},
			&SourceRange{
				Filename: "somewhere_else.tf",
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d:%s", i, tc.Diag.Description()), func(t *testing.T) {
			var diags Diagnostics
			diags = diags.Append(tc.Diag)
			gotDiags := diags.InConfigBody(f.Body)
			gotRange := gotDiags[0].Source().Subject

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
