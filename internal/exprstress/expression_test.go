package exprstress

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestExprLiteral(t *testing.T) {
	tests := []struct {
		Value        cty.Value
		WantSource   string
		WantExpected Expected
	}{
		// exprLiteral only works with the subset of values that
		// hclwrite.TokensForValue can faithfully represent. Some
		// of the values excluded from that (and thus intentionally
		// not tested below) are:
		// - null values with any type other than DynamicPseudoType
		// - lists and maps (they become tuples and objects)
		// - unknown values (there is no literal syntax for those)
		// - sensitive values (there is no literal syntax for those)
		{
			cty.NullVal(cty.DynamicPseudoType),
			`null`,
			Expected{
				Type: cty.DynamicPseudoType,
				Mode: NullValue,
			},
		},
		{
			cty.StringVal("hello"),
			`"hello"`,
			Expected{
				Type: cty.String,
				Mode: SpecifiedValue,
			},
		},
		{
			cty.NumberIntVal(1),
			`1`,
			Expected{
				Type: cty.Number,
				Mode: SpecifiedValue,
			},
		},
		{
			cty.True,
			`true`,
			Expected{
				Type: cty.Bool,
				Mode: SpecifiedValue,
			},
		},
		{
			cty.EmptyTupleVal,
			`[]`,
			Expected{
				Type: cty.EmptyTuple,
				Mode: SpecifiedValue,
			},
		},
		{
			cty.TupleVal([]cty.Value{cty.True}),
			`[true]`,
			Expected{
				Type: cty.Tuple([]cty.Type{cty.Bool}),
				Mode: SpecifiedValue,
			},
		},
		{
			cty.TupleVal([]cty.Value{cty.NullVal(cty.DynamicPseudoType)}),
			`[null]`,
			Expected{
				Type: cty.Tuple([]cty.Type{cty.DynamicPseudoType}),
				Mode: SpecifiedValue, // top-level is specified, even though element is null
			},
		},
		{
			cty.EmptyObjectVal,
			`{}`,
			Expected{
				Type: cty.EmptyObject,
				Mode: SpecifiedValue,
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{"boop": cty.True}),
			`{
  boop = true
}`,
			Expected{
				Type: cty.Object(map[string]cty.Type{"boop": cty.Bool}),
				Mode: SpecifiedValue,
			},
		},
		{
			cty.ObjectVal(map[string]cty.Value{"blorp": cty.NullVal(cty.DynamicPseudoType)}),
			`{
  blorp = null
}`,
			Expected{
				Type: cty.Object(map[string]cty.Type{"blorp": cty.DynamicPseudoType}),
				Mode: SpecifiedValue, // top-level is specified, even though element is null
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Value.GoString(), func(t *testing.T) {
			expr := &exprLiteral{
				Value: test.Value,
			}
			var buf strings.Builder
			expr.BuildSource(&buf)
			gotSource := buf.String()
			gotExpected := expr.ExpectedResult()

			if got, want := gotSource, test.WantSource; got != want {
				t.Errorf("wrong source code\ngot:  %s\nwant: %s", got, want)
			}
			if diff := cmp.Diff(test.WantExpected, gotExpected, ctydebug.CmpOptions); diff != "" {
				t.Errorf("wrong expected result\n%s", diff)
			}
		})
	}
}
