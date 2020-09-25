package objchange

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestLongestCommonSubsequence(t *testing.T) {
	tests := []struct {
		xs   []cty.Value
		ys   []cty.Value
		want []cty.Value
	}{
		{
			[]cty.Value{},
			[]cty.Value{},
			[]cty.Value{},
		},
		{
			[]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2)},
			[]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2)},
			[]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2)},
		},
		{
			[]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2)},
			[]cty.Value{cty.NumberIntVal(3), cty.NumberIntVal(4)},
			[]cty.Value{},
		},
		{
			[]cty.Value{cty.NumberIntVal(2)},
			[]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2)},
			[]cty.Value{cty.NumberIntVal(2)},
		},
		{
			[]cty.Value{cty.NumberIntVal(1)},
			[]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2)},
			[]cty.Value{cty.NumberIntVal(1)},
		},
		{
			[]cty.Value{cty.NumberIntVal(2), cty.NumberIntVal(1)},
			[]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2)},
			[]cty.Value{cty.NumberIntVal(1)}, // arbitrarily selected 1; 2 would also be valid
		},
		{
			[]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2), cty.NumberIntVal(3), cty.NumberIntVal(4)},
			[]cty.Value{cty.NumberIntVal(2), cty.NumberIntVal(4), cty.NumberIntVal(5)},
			[]cty.Value{cty.NumberIntVal(2), cty.NumberIntVal(4)},
		},
		{
			[]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2), cty.NumberIntVal(3), cty.NumberIntVal(4)},
			[]cty.Value{cty.NumberIntVal(4), cty.NumberIntVal(2), cty.NumberIntVal(5)},
			[]cty.Value{cty.NumberIntVal(4)}, // 2 would also be valid
		},
		{
			[]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2), cty.NumberIntVal(3), cty.NumberIntVal(5)},
			[]cty.Value{cty.NumberIntVal(2), cty.NumberIntVal(4), cty.NumberIntVal(5)},
			[]cty.Value{cty.NumberIntVal(2), cty.NumberIntVal(5)},
		},

		// unknowns never compare as equal
		{
			[]cty.Value{cty.NumberIntVal(1), cty.UnknownVal(cty.Number), cty.NumberIntVal(3)},
			[]cty.Value{cty.NumberIntVal(1), cty.UnknownVal(cty.Number), cty.NumberIntVal(3)},
			[]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(3)},
		},
		{
			[]cty.Value{cty.UnknownVal(cty.Number)},
			[]cty.Value{cty.UnknownVal(cty.Number)},
			[]cty.Value{},
		},

		// marked values
		{
			[]cty.Value{cty.NumberIntVal(1).Mark("foo"), cty.NumberIntVal(2).Mark("foo"), cty.NumberIntVal(3)},
			[]cty.Value{cty.NumberIntVal(1).Mark("foo"), cty.NumberIntVal(2).Mark("foo")},
			[]cty.Value{cty.NumberIntVal(1).Mark("foo"), cty.NumberIntVal(2).Mark("foo")},
		},
		{
			[]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2).Mark("foo"), cty.NumberIntVal(3)},
			[]cty.Value{cty.NumberIntVal(2), cty.NumberIntVal(3)},
			[]cty.Value{cty.NumberIntVal(3)},
		},
		{
			[]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2).Mark("foo")},
			[]cty.Value{cty.NumberIntVal(2)},
			[]cty.Value{},
		},
		{
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{"a": cty.StringVal("x").Mark("sensitive")}),
				cty.MapVal(map[string]cty.Value{"b": cty.StringVal("y")}),
			},
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{"a": cty.StringVal("x").Mark("sensitive")}),
				cty.MapVal(map[string]cty.Value{"b": cty.StringVal("y")}),
				cty.MapVal(map[string]cty.Value{"c": cty.StringVal("z")}),
			},
			[]cty.Value{
				cty.MapVal(map[string]cty.Value{"a": cty.StringVal("x").Mark("sensitive")}),
				cty.MapVal(map[string]cty.Value{"b": cty.StringVal("y")}),
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v,%#v", test.xs, test.ys), func(t *testing.T) {
			got := LongestCommonSubsequence(test.xs, test.ys)

			wrong := func() {
				t.Fatalf(
					"wrong result\nX:    %#v\nY:    %#v\ngot:  %#v\nwant: %#v",
					test.xs, test.ys, got, test.want,
				)
			}

			if len(got) != len(test.want) {
				wrong()
			}

			for i := range got {
				if got[i] == cty.NilVal {
					wrong()
				}
				if !got[i].RawEquals(test.want[i]) {
					wrong()
				}
			}
		})
	}
}
