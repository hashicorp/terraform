// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestParseRemoveTarget(t *testing.T) {
	tests := []struct {
		Input   string
		Want    ConfigMoveable
		WantErr string
	}{
		{
			`test_instance.bar`,
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "test_instance",
					Name: "bar",
				},
			},
			``,
		},
		{
			`module.foo.test_instance.bar`,
			ConfigResource{
				Module: []string{"foo"},
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "test_instance",
					Name: "bar",
				},
			},
			``,
		},
		{
			`module.foo.module.baz.test_instance.bar`,
			ConfigResource{
				Module: []string{"foo", "baz"},
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "test_instance",
					Name: "bar",
				},
			},
			``,
		},
		{
			`data.test_ds.moo`,
			nil,
			`Data source address not allowed: Data sources are never destroyed, so they are not valid targets of removed blocks. To remove the data source from state, remove the data source block from configuration.`,
		},
		{
			`module.foo.data.test_ds.noo`,
			nil,
			`Data source address not allowed: Data sources are never destroyed, so they are not valid targets of removed blocks. To remove the data source from state, remove the data source block from configuration.`,
		},
		{
			`test_instance.foo[0]`,
			nil,
			`Resource instance keys not allowed: Resource address must be a resource (e.g. "test_instance.foo"), not a resource instance (e.g. "test_instance.foo[1]").`,
		},
		{
			`module.foo[0].test_instance.bar`,
			nil,
			`Module instance keys not allowed: Module address must be a module (e.g. "module.foo"), not a module instance (e.g. "module.foo[1]").`,
		},
		{
			`module.foo.test_instance.bar[0]`,
			nil,
			`Resource instance keys not allowed: Resource address must be a resource (e.g. "test_instance.foo"), not a resource instance (e.g. "test_instance.foo[1]").`,
		},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(test.Input), "", hcl.InitialPos)
			if hclDiags.HasErrors() {
				// We're not trying to test the HCL parser here, so any
				// failures at this point are likely to be bugs in the
				// test case itself.
				t.Fatalf("syntax error: %s", hclDiags.Error())
			}

			remT, diags := ParseRemoveTarget(traversal)

			switch {
			case test.WantErr != "":
				if !diags.HasErrors() {
					t.Fatalf("unexpected success\nwant error: %s", test.WantErr)
				}
				gotErr := diags.Err().Error()
				if gotErr != test.WantErr {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", gotErr, test.WantErr)
				}
			default:
				if diags.HasErrors() {
					t.Fatalf("unexpected error: %s", diags.Err().Error())
				}
				if diff := cmp.Diff(test.Want, remT.RelSubject); diff != "" {
					t.Errorf("wrong result\n%s", diff)
				}
			}
		})
	}
}
