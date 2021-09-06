package cty_diff

import (
	"encoding/json"
	"testing"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func TestValueDiffCost(t *testing.T) {
	expectCost(
		t,
		cty.True,
		cty.True,
		1,
	)
	expectCost(
		t,
		cty.True,
		cty.ListVal([]cty.Value{cty.True, cty.True, cty.True}),
		1,
	)
	expectCost(
		t,
		cty.ListVal([]cty.Value{cty.True, cty.True}),
		cty.ListVal([]cty.Value{cty.True, cty.True, cty.True}),
		12, // (1 + 2) * (1 + 3)
	)
	expectCost(
		t,
		cty.ListValEmpty(cty.Bool),
		cty.ListVal([]cty.Value{cty.True, cty.True}),
		3, // (1 + 0) * (1 + 2)
	)
	expectCost(
		t,
		cty.ListValEmpty(cty.List(cty.List(cty.Bool))),
		cty.ListVal([]cty.Value{
			cty.ListVal([]cty.Value{
				cty.ListVal([]cty.Value{
					cty.True,
				}),
			}),
		}),
		4, // (1 + 0) * (1 + (1 + (1 + 1)))
	)
}

func expectCost(t *testing.T, a, b cty.Value, expectedCost float32) {
	actualCost := ValueDiffCost(a, b)
	if actualCost != expectedCost {
		t.Errorf("Unexpected cost %f; wanted %f\nLeft : %s\nRight: %s",
			actualCost, expectedCost, ctyToJson(a), ctyToJson(b))
	}
}

func ctyToJson(v cty.Value) string {
	jsonBytes, _ := json.Marshal(ctyjson.SimpleJSONValue{v})
	return string(jsonBytes)
}
