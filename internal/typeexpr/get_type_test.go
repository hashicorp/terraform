package typeexpr

import (
	"testing"

	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/json"
	"github.com/zclconf/go-cty/cty"
)

func TestGetType(t *testing.T) {
	tests := []struct {
		Source     string
		Constraint bool
		Want       cty.Type
		WantError  string
	}{
		// keywords
		{
			`bool`,
			false,
			cty.Bool,
			"",
		},
		{
			`number`,
			false,
			cty.Number,
			"",
		},
		{
			`string`,
			false,
			cty.String,
			"",
		},
		{
			`any`,
			false,
			cty.DynamicPseudoType,
			`The keyword "any" cannot be used in this type specification: an exact type is required.`,
		},
		{
			`any`,
			true,
			cty.DynamicPseudoType,
			"",
		},
		{
			`list`,
			false,
			cty.DynamicPseudoType,
			"The list type constructor requires one argument specifying the element type.",
		},
		{
			`map`,
			false,
			cty.DynamicPseudoType,
			"The map type constructor requires one argument specifying the element type.",
		},
		{
			`set`,
			false,
			cty.DynamicPseudoType,
			"The set type constructor requires one argument specifying the element type.",
		},
		{
			`object`,
			false,
			cty.DynamicPseudoType,
			"The object type constructor requires one argument specifying the attribute types and values as a map.",
		},
		{
			`tuple`,
			false,
			cty.DynamicPseudoType,
			"The tuple type constructor requires one argument specifying the element types as a list.",
		},

		// constructors
		{
			`bool()`,
			false,
			cty.DynamicPseudoType,
			`Primitive type keyword "bool" does not expect arguments.`,
		},
		{
			`number()`,
			false,
			cty.DynamicPseudoType,
			`Primitive type keyword "number" does not expect arguments.`,
		},
		{
			`string()`,
			false,
			cty.DynamicPseudoType,
			`Primitive type keyword "string" does not expect arguments.`,
		},
		{
			`any()`,
			false,
			cty.DynamicPseudoType,
			`Primitive type keyword "any" does not expect arguments.`,
		},
		{
			`any()`,
			true,
			cty.DynamicPseudoType,
			`Primitive type keyword "any" does not expect arguments.`,
		},
		{
			`list(string)`,
			false,
			cty.List(cty.String),
			``,
		},
		{
			`set(string)`,
			false,
			cty.Set(cty.String),
			``,
		},
		{
			`map(string)`,
			false,
			cty.Map(cty.String),
			``,
		},
		{
			`list()`,
			false,
			cty.DynamicPseudoType,
			`The list type constructor requires one argument specifying the element type.`,
		},
		{
			`list(string, string)`,
			false,
			cty.DynamicPseudoType,
			`The list type constructor requires one argument specifying the element type.`,
		},
		{
			`list(any)`,
			false,
			cty.List(cty.DynamicPseudoType),
			`The keyword "any" cannot be used in this type specification: an exact type is required.`,
		},
		{
			`list(any)`,
			true,
			cty.List(cty.DynamicPseudoType),
			``,
		},
		{
			`object({})`,
			false,
			cty.EmptyObject,
			``,
		},
		{
			`object({name=string})`,
			false,
			cty.Object(map[string]cty.Type{"name": cty.String}),
			``,
		},
		{
			`object({"name"=string})`,
			false,
			cty.EmptyObject,
			`Object constructor map keys must be attribute names.`,
		},
		{
			`object({name=nope})`,
			false,
			cty.Object(map[string]cty.Type{"name": cty.DynamicPseudoType}),
			`The keyword "nope" is not a valid type specification.`,
		},
		{
			`object()`,
			false,
			cty.DynamicPseudoType,
			`The object type constructor requires one argument specifying the attribute types and values as a map.`,
		},
		{
			`object(string)`,
			false,
			cty.DynamicPseudoType,
			`Object type constructor requires a map whose keys are attribute names and whose values are the corresponding attribute types.`,
		},
		{
			`tuple([])`,
			false,
			cty.EmptyTuple,
			``,
		},
		{
			`tuple([string, bool])`,
			false,
			cty.Tuple([]cty.Type{cty.String, cty.Bool}),
			``,
		},
		{
			`tuple([nope])`,
			false,
			cty.Tuple([]cty.Type{cty.DynamicPseudoType}),
			`The keyword "nope" is not a valid type specification.`,
		},
		{
			`tuple()`,
			false,
			cty.DynamicPseudoType,
			`The tuple type constructor requires one argument specifying the element types as a list.`,
		},
		{
			`tuple(string)`,
			false,
			cty.DynamicPseudoType,
			`Tuple type constructor requires a list of element types.`,
		},
		{
			`shwoop(string)`,
			false,
			cty.DynamicPseudoType,
			`Keyword "shwoop" is not a valid type constructor.`,
		},
		{
			`list("string")`,
			false,
			cty.List(cty.DynamicPseudoType),
			`A type specification is either a primitive type keyword (bool, number, string) or a complex type constructor call, like list(string).`,
		},

		// More interesting combinations
		{
			`list(object({}))`,
			false,
			cty.List(cty.EmptyObject),
			``,
		},
		{
			`list(map(tuple([])))`,
			false,
			cty.List(cty.Map(cty.EmptyTuple)),
			``,
		},
	}

	for _, test := range tests {
		t.Run(test.Source, func(t *testing.T) {
			expr, diags := hclsyntax.ParseExpression([]byte(test.Source), "", hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				t.Fatalf("failed to parse: %s", diags)
			}

			got, diags := getType(expr, test.Constraint)
			if test.WantError == "" {
				for _, diag := range diags {
					t.Error(diag)
				}
			} else {
				found := false
				for _, diag := range diags {
					t.Log(diag)
					if diag.Severity == hcl.DiagError && diag.Detail == test.WantError {
						found = true
					}
				}
				if !found {
					t.Errorf("missing expected error detail message: %s", test.WantError)
				}
			}

			if !got.Equals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestGetTypeJSON(t *testing.T) {
	// We have fewer test cases here because we're mainly exercising the
	// extra indirection in the JSON syntax package, which ultimately calls
	// into the native syntax parser (which we tested extensively in
	// TestGetType).
	tests := []struct {
		Source     string
		Constraint bool
		Want       cty.Type
		WantError  string
	}{
		{
			`{"expr":"bool"}`,
			false,
			cty.Bool,
			"",
		},
		{
			`{"expr":"list(bool)"}`,
			false,
			cty.List(cty.Bool),
			"",
		},
		{
			`{"expr":"list"}`,
			false,
			cty.DynamicPseudoType,
			"The list type constructor requires one argument specifying the element type.",
		},
	}

	for _, test := range tests {
		t.Run(test.Source, func(t *testing.T) {
			file, diags := json.Parse([]byte(test.Source), "")
			if diags.HasErrors() {
				t.Fatalf("failed to parse: %s", diags)
			}

			type TestContent struct {
				Expr hcl.Expression `hcl:"expr"`
			}
			var content TestContent
			diags = gohcl.DecodeBody(file.Body, nil, &content)
			if diags.HasErrors() {
				t.Fatalf("failed to decode: %s", diags)
			}

			got, diags := getType(content.Expr, test.Constraint)
			if test.WantError == "" {
				for _, diag := range diags {
					t.Error(diag)
				}
			} else {
				found := false
				for _, diag := range diags {
					t.Log(diag)
					if diag.Severity == hcl.DiagError && diag.Detail == test.WantError {
						found = true
					}
				}
				if !found {
					t.Errorf("missing expected error detail message: %s", test.WantError)
				}
			}

			if !got.Equals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
