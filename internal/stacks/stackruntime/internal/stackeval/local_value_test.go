// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/zclconf/go-cty/cty"
)

func TestLocalValueValue(t *testing.T) {
	ctx := context.Background()
	cfg := testStackConfig(t, "local_value", "basics")

	tests := map[string]struct {
		LocalName string
		WantVal   cty.Value
	}{
		"name": {
			LocalName: "name",
			WantVal:   cty.StringVal("jackson"),
		},
		"childName": {
			LocalName: "childName",
			WantVal:   cty.StringVal("outputted-child of jackson"),
		},
		"functional": {
			LocalName: "functional",
			WantVal:   cty.StringVal("Hello, Ander!"),
		},
		"mappy": {
			LocalName: "mappy",
			WantVal: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("jackson"),
				"age":  cty.NumberIntVal(30),
			}),
		},
		"listy": {
			LocalName: "listy",
			WantVal: cty.TupleVal([]cty.Value{
				cty.StringVal("jackson"),
				cty.NumberIntVal(30),
			}),
		},
		"booleany": {
			LocalName: "booleany",
			WantVal:   cty.BoolVal(true),
		},
		"conditiony": {
			LocalName: "conditiony",
			WantVal:   cty.StringVal("true"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			main := testEvaluator(t, testEvaluatorOpts{
				Config: cfg,
			})

			promising.MainTask(ctx, func(ctx context.Context) (struct{}, error) {
				mainStack := main.MainStack(ctx)
				rootVal := mainStack.LocalValue(ctx, stackaddrs.LocalValue{Name: test.LocalName})
				got, diags := rootVal.CheckValue(ctx, InspectPhase)

				if diags.HasErrors() {
					t.Errorf("unexpected errors\n%s", diags.Err().Error())
				}

				if got.Equals(test.WantVal).False() {
					t.Errorf("got %s, want %s", got, test.WantVal)
				}

				return struct{}{}, nil
			})
		})
	}
}
