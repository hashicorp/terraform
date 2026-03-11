// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/states"
)

func TestNodeLocalExecute(t *testing.T) {
	tests := []struct {
		Value string
		Want  cty.Value
		Err   bool
	}{
		{
			"hello!",
			cty.StringVal("hello!"),
			false,
		},
		{
			"",
			cty.StringVal(""),
			false,
		},
		{
			"Hello, ${local.foo}",
			cty.DynamicVal,
			true, // self-referencing
		},
	}

	for _, test := range tests {
		t.Run(test.Value, func(t *testing.T) {
			expr, diags := hclsyntax.ParseTemplate([]byte(test.Value), "", hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				t.Fatal(diags.Error())
			}

			localAddr := addrs.LocalValue{Name: "foo"}.Absolute(addrs.RootModuleInstance)
			n := &NodeLocal{
				Addr: localAddr,
				Config: &configs.Local{
					Expr: expr,
				},
			}
			ctx := &MockEvalContext{
				StateState:       states.NewState().SyncWrapper(),
				NamedValuesState: namedvals.NewState(),

				EvaluateExprResult: test.Want,
			}

			err := n.Execute(ctx, walkApply)
			if (err != nil) != test.Err {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				} else {
					t.Errorf("successful Eval; want error")
				}
			}

			if !ctx.NamedValues().HasLocalValue(localAddr) {
				t.Fatalf("no value for %s", localAddr)
			}
			got := ctx.NamedValues().GetLocalValue(localAddr)
			want := test.Want
			if !want.RawEquals(got) {
				t.Errorf("wrong value for %s\ngot:  %#v\nwant: %#v", localAddr, got, want)
			}
		})
	}

}
