// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/hcl2shim"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/states"
)

func TestNodeLocalExecute(t *testing.T) {
	tests := []struct {
		Value string
		Want  interface{}
		Err   bool
	}{
		{
			"hello!",
			"hello!",
			false,
		},
		{
			"",
			"",
			false,
		},
		{
			"Hello, ${local.foo}",
			nil,
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

				EvaluateExprResult: hcl2shim.HCL2ValueFromConfigValue(test.Want),
			}

			err := n.Execute(ctx, walkApply)
			if (err != nil) != test.Err {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				} else {
					t.Errorf("successful Eval; want error")
				}
			}

			if test.Err {
				if ctx.NamedValues().HasLocalValue(localAddr) {
					t.Errorf("have value for %s, but wanted none", localAddr)
				}
			} else {
				if !ctx.NamedValues().HasLocalValue(localAddr) {
					t.Fatalf("no value for %s", localAddr)
				}
				got := ctx.NamedValues().GetLocalValue(localAddr)
				want := hcl2shim.HCL2ValueFromConfigValue(test.Want)
				if !want.RawEquals(got) {
					t.Errorf("wrong value for %s\ngot:  %#v\nwant: %#v", localAddr, got, want)
				}
			}
		})
	}

}
