package typeexpr

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/json"
	"github.com/zclconf/go-cty/cty"
)

var (
	typeComparer = cmp.Comparer(cty.Type.Equals)
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
			`Type constraint keyword "any" does not expect arguments.`,
		},
		{
			`any()`,
			true,
			cty.DynamicPseudoType,
			`Type constraint keyword "any" does not expect arguments.`,
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

		// Optional modifier
		{
			`object({name=string,age=optional(number)})`,
			true,
			cty.ObjectWithOptionalAttrs(map[string]cty.Type{
				"name": cty.String,
				"age":  cty.Number,
			}, []string{"age"}),
			``,
		},
		{
			`object({name=string,meta=optional(any)})`,
			true,
			cty.ObjectWithOptionalAttrs(map[string]cty.Type{
				"name": cty.String,
				"meta": cty.DynamicPseudoType,
			}, []string{"meta"}),
			``,
		},
		{
			`object({name=string,age=optional(number)})`,
			false,
			cty.Object(map[string]cty.Type{
				"name": cty.String,
				"age":  cty.Number,
			}),
			`Optional attribute modifier is only for type constraints, not for exact types.`,
		},
		{
			`object({name=string,meta=optional(any)})`,
			false,
			cty.Object(map[string]cty.Type{
				"name": cty.String,
				"meta": cty.DynamicPseudoType,
			}),
			`Optional attribute modifier is only for type constraints, not for exact types.`,
		},
		{
			`object({name=string,meta=optional()})`,
			true,
			cty.Object(map[string]cty.Type{
				"name": cty.String,
			}),
			`Optional attribute modifier requires the attribute type as its argument.`,
		},
		{
			`object({name=string,meta=optional(string, "hello")})`,
			true,
			cty.Object(map[string]cty.Type{
				"name": cty.String,
				"meta": cty.String,
			}),
			`Optional attribute modifier expects only one argument: the attribute type.`,
		},
		{
			`optional(string)`,
			false,
			cty.DynamicPseudoType,
			`Keyword "optional" is valid only as a modifier for object type attributes.`,
		},
		{
			`optional`,
			false,
			cty.DynamicPseudoType,
			`The keyword "optional" is not a valid type specification.`,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s (constraint=%v)", test.Source, test.Constraint), func(t *testing.T) {
			expr, diags := hclsyntax.ParseExpression([]byte(test.Source), "", hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				t.Fatalf("failed to parse: %s", diags)
			}

			got, _, diags := getType(expr, test.Constraint, false)
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

			got, _, diags := getType(content.Expr, test.Constraint, false)
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

func TestGetTypeDefaults(t *testing.T) {
	tests := []struct {
		Source    string
		Want      *Defaults
		WantError string
	}{
		// primitive types have nil defaults
		{
			`bool`,
			nil,
			"",
		},
		{
			`number`,
			nil,
			"",
		},
		{
			`string`,
			nil,
			"",
		},
		{
			`any`,
			nil,
			"",
		},

		// complex structures with no defaults have nil defaults
		{
			`map(string)`,
			nil,
			"",
		},
		{
			`set(number)`,
			nil,
			"",
		},
		{
			`tuple([number, string])`,
			nil,
			"",
		},
		{
			`object({ a = string, b = number })`,
			nil,
			"",
		},
		{
			`map(list(object({ a = string, b = optional(number) })))`,
			nil,
			"",
		},

		// object optional attribute with defaults
		{
			`object({ a = string, b = optional(number, 5) })`,
			&Defaults{
				Type: cty.ObjectWithOptionalAttrs(map[string]cty.Type{
					"a": cty.String,
					"b": cty.Number,
				}, []string{"b"}),
				DefaultValues: map[string]cty.Value{
					"b": cty.NumberIntVal(5),
				},
			},
			"",
		},

		// nested defaults
		{
			`object({ a = optional(object({ b = optional(number, 5) }), {}) })`,
			&Defaults{
				Type: cty.ObjectWithOptionalAttrs(map[string]cty.Type{
					"a": cty.ObjectWithOptionalAttrs(map[string]cty.Type{
						"b": cty.Number,
					}, []string{"b"}),
				}, []string{"a"}),
				DefaultValues: map[string]cty.Value{
					"a": cty.EmptyObjectVal,
				},
				Children: map[string]*Defaults{
					"a": {
						Type: cty.ObjectWithOptionalAttrs(map[string]cty.Type{
							"b": cty.Number,
						}, []string{"b"}),
						DefaultValues: map[string]cty.Value{
							"b": cty.NumberIntVal(5),
						},
					},
				},
			},
			"",
		},

		// collections of objects with defaults
		{
			`map(object({ a = string, b = optional(number, 5) }))`,
			&Defaults{
				Type: cty.Map(cty.ObjectWithOptionalAttrs(map[string]cty.Type{
					"a": cty.String,
					"b": cty.Number,
				}, []string{"b"})),
				Children: map[string]*Defaults{
					"": {
						Type: cty.ObjectWithOptionalAttrs(map[string]cty.Type{
							"a": cty.String,
							"b": cty.Number,
						}, []string{"b"}),
						DefaultValues: map[string]cty.Value{
							"b": cty.NumberIntVal(5),
						},
					},
				},
			},
			"",
		},
		{
			`list(object({ a = string, b = optional(number, 5) }))`,
			&Defaults{
				Type: cty.List(cty.ObjectWithOptionalAttrs(map[string]cty.Type{
					"a": cty.String,
					"b": cty.Number,
				}, []string{"b"})),
				Children: map[string]*Defaults{
					"": {
						Type: cty.ObjectWithOptionalAttrs(map[string]cty.Type{
							"a": cty.String,
							"b": cty.Number,
						}, []string{"b"}),
						DefaultValues: map[string]cty.Value{
							"b": cty.NumberIntVal(5),
						},
					},
				},
			},
			"",
		},
		{
			`set(object({ a = string, b = optional(number, 5) }))`,
			&Defaults{
				Type: cty.Set(cty.ObjectWithOptionalAttrs(map[string]cty.Type{
					"a": cty.String,
					"b": cty.Number,
				}, []string{"b"})),
				Children: map[string]*Defaults{
					"": {
						Type: cty.ObjectWithOptionalAttrs(map[string]cty.Type{
							"a": cty.String,
							"b": cty.Number,
						}, []string{"b"}),
						DefaultValues: map[string]cty.Value{
							"b": cty.NumberIntVal(5),
						},
					},
				},
			},
			"",
		},

		// tuples containing objects with defaults work differently from
		// collections
		{
			`tuple([string, bool, object({ a = string, b = optional(number, 5) })])`,
			&Defaults{
				Type: cty.Tuple([]cty.Type{
					cty.String,
					cty.Bool,
					cty.ObjectWithOptionalAttrs(map[string]cty.Type{
						"a": cty.String,
						"b": cty.Number,
					}, []string{"b"}),
				}),
				Children: map[string]*Defaults{
					"2": {
						Type: cty.ObjectWithOptionalAttrs(map[string]cty.Type{
							"a": cty.String,
							"b": cty.Number,
						}, []string{"b"}),
						DefaultValues: map[string]cty.Value{
							"b": cty.NumberIntVal(5),
						},
					},
				},
			},
			"",
		},

		// incompatible default value causes an error
		{
			`object({ a = optional(string, "hello"), b = optional(number, true) })`,
			&Defaults{
				Type: cty.ObjectWithOptionalAttrs(map[string]cty.Type{
					"a": cty.String,
					"b": cty.Number,
				}, []string{"a", "b"}),
				DefaultValues: map[string]cty.Value{
					"a": cty.StringVal("hello"),
				},
			},
			"This default value is not compatible with the attribute's type constraint: number required.",
		},

		// Too many arguments
		{
			`object({name=string,meta=optional(string, "hello", "world")})`,
			nil,
			`Optional attribute modifier expects at most two arguments: the attribute type, and a default value.`,
		},
	}

	for _, test := range tests {
		t.Run(test.Source, func(t *testing.T) {
			expr, diags := hclsyntax.ParseExpression([]byte(test.Source), "", hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				t.Fatalf("failed to parse: %s", diags)
			}

			_, got, diags := getType(expr, true, true)
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

			if !cmp.Equal(test.Want, got, valueComparer, typeComparer) {
				t.Errorf("wrong result\n%s", cmp.Diff(test.Want, got, valueComparer, typeComparer))
			}
		})
	}
}
