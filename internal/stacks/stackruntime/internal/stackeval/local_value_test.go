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
		// "childName": {
		// 	LocalName: "childName",
		// 	WantVal:   cty.StringVal("foo"),
		// },
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

				if got != test.WantVal {
					t.Errorf("got %s, want %s", got, test.WantVal)
				}

				return struct{}{}, nil
			})
		})
	}

	// main := testEvaluator(t, testEvaluatorOpts{
	// 	Config: cfg,
	// })
	//
	// promising.MainTask(ctx, func(ctx context.Context) (struct{}, error) {
	// 	mainStack := main.MainStack(ctx)
	// 	rootVal := mainStack.LocalValue(ctx, stackaddrs.LocalValue{Name: "name"})
	// 	got, diags := rootVal.CheckValue(ctx, InspectPhase)
	//
	// 	if diags.HasErrors() {
	// 		t.Errorf("unexpected errors\n%s", diags.Err().Error())
	// 	}
	//
	// 	want := cty.StringVal("parent")
	// 	if got != want {
	// 		t.Errorf("got %s, want %s", got, want)
	// 	}
	//
	// 	return struct{}{}, nil
	// })
}
