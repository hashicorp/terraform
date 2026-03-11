// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func TestPartialExpandedResourceIsTargetedBy(t *testing.T) {

	tcs := []struct {
		per    string
		target string
		want   bool
	}{
		{
			"test.a",
			"test.a",
			true,
		},
		{
			"test.a",
			"test.a[0]",
			true,
		},
		{
			"test.a[*]",
			"test.a",
			true,
		},
		{
			"test.a[*]",
			"test.a[0]",
			true,
		},
		{
			"test.a[*]",
			"test.a[\"key\"]",
			true,
		},
		{
			"module.mod.test.a",
			"module.mod.test.a",
			true,
		},
		{
			"module.mod[1].test.a",
			"module.mod[0].test.a",
			false,
		},
		{
			"module.mod.test.a[*]",
			"module.mod.test.a",
			true,
		},
		{
			"module.mod.test.a[*]",
			"module.mod.test.a[0]",
			true,
		},
		{
			"module.mod.test.a[*]",
			"module.mod.test.a[\"key\"]",
			true,
		},
		{
			"module.mod.test.a[*]",
			"module.mod[0].test.a",
			false,
		},
		{
			"module.mod[1].test.a[*]",
			"module.mod[\"key\"].test.a[0]",
			false,
		},
		{
			"module.mod[*].test.a",
			"module.mod.test.a",
			true,
		},
		{
			"module.mod[*].test.a",
			"module.mod.test.a[0]",
			true,
		},
		{
			"module.mod[*].test.a",
			"module.mod[0].test.a",
			true,
		},
		{
			"module.mod[*].test.a",
			"module.mod[\"key\"].test.a",
			true,
		},
	}

	for _, tc := range tcs {
		t.Run(fmt.Sprintf("PartialResource(%q).IsTargetedBy(%q)", tc.per, tc.target), func(t *testing.T) {
			per := mustParsePartialResourceInstanceStr(tc.per).PartialResource()
			target := mustParseTarget(tc.target)

			got := per.IsTargetedBy(target)
			if got != tc.want {
				t.Errorf("PartialResource(%q).IsTargetedBy(%q): got %v; want %v", tc.per, tc.target, got, tc.want)
			}
		})
	}

}

func TestParsePartialExpandedModule(t *testing.T) {

	// these functions are a bit weird, as the normal parsing supported by
	// HCL can't put unknown values into the instance keys. So we need to
	// build the traversals in the same way the thing that is calling these
	// functions does.

	tcs := []struct {
		traversal func(t *testing.T) (string, hcl.Traversal)
		want      PartialExpandedModule
		remain    int
	}{
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.mod"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				return addr, traversal
			},
			want: PartialExpandedModule{
				expandedPrefix: ModuleInstance{
					{
						Name: "mod",
					},
				},
			},
			remain: 0,
		},
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.mod[0]"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				// Hack the key into an unknown value.
				traversal[2] = hcl.TraverseIndex{
					Key: cty.UnknownVal(cty.Number),
				}
				return "module.mod[*]", traversal
			},
			want: PartialExpandedModule{
				unexpandedSuffix: Module{
					"mod",
				},
			},
			remain: 0,
		},
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.child.module.grandchild"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				return addr, traversal
			},
			want: PartialExpandedModule{
				expandedPrefix: ModuleInstance{
					{
						Name: "child",
					},
					{
						Name: "grandchild",
					},
				},
			},
			remain: 0,
		},
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.child[0].module.grandchild"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				return addr, traversal
			},
			want: PartialExpandedModule{
				expandedPrefix: ModuleInstance{
					{
						Name:        "child",
						InstanceKey: IntKey(0),
					},
					{
						Name: "grandchild",
					},
				},
			},
			remain: 0,
		},
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.child[0].module.grandchild"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				traversal[2] = hcl.TraverseIndex{
					Key: cty.UnknownVal(cty.Number),
				}
				return "module.child[*].module.grandchild", traversal
			},
			want: PartialExpandedModule{
				unexpandedSuffix: Module{
					"child",
					"grandchild",
				},
			},
			remain: 0,
		},
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.child.module.grandchild[0]"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				traversal[4] = hcl.TraverseIndex{
					Key: cty.UnknownVal(cty.Number),
				}
				return "module.child.module.grandchild[*]", traversal
			},
			want: PartialExpandedModule{
				expandedPrefix: ModuleInstance{
					{
						Name: "child",
					},
				},
				unexpandedSuffix: Module{
					"grandchild",
				},
			},
			remain: 0,
		},
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.child.module.grandchild[0].resource_type.resource_name"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				traversal[4] = hcl.TraverseIndex{
					Key: cty.UnknownVal(cty.Number),
				}
				return "module.child.module.grandchild[*].resource_type.resource_name", traversal
			},
			want: PartialExpandedModule{
				expandedPrefix: ModuleInstance{
					{
						Name: "child",
					},
				},
				unexpandedSuffix: Module{
					"grandchild",
				},
			},
			remain: 2,
		},
	}

	for _, tc := range tcs {
		addr, traversal := tc.traversal(t)
		t.Run(addr, func(t *testing.T) {
			module, rest, diags := ParsePartialExpandedModule(traversal)
			if len(diags) > 0 {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}

			if !module.expandedPrefix.Equal(tc.want.expandedPrefix) {
				t.Errorf("got expandedPrefix %v; want %v", module.expandedPrefix, tc.want.expandedPrefix)
			}
			if !module.unexpandedSuffix.Equal(tc.want.unexpandedSuffix) {
				t.Errorf("got unexpandedSuffix %v; want %v", module.unexpandedSuffix, tc.want.unexpandedSuffix)
			}
			if len(rest) != tc.remain {
				t.Errorf("got %d remaining traversals; want %d", len(rest), tc.remain)
			}
		})
	}

}

func TestParsePartialExpandedResource(t *testing.T) {

	tcs := []struct {
		addr   string
		want   PartialExpandedResource
		remain int
	}{
		{
			addr: "resource_type.resource_name",
			want: PartialExpandedResource{
				resource: Resource{
					Mode: ManagedResourceMode,
					Type: "resource_type",
					Name: "resource_name",
				},
			},
			remain: 0,
		},
		{
			addr: "module.mod.resource_type.resource_name",
			want: PartialExpandedResource{
				module: PartialExpandedModule{
					expandedPrefix: ModuleInstance{
						{
							Name: "mod",
						},
					},
				},
				resource: Resource{
					Mode: ManagedResourceMode,
					Type: "resource_type",
					Name: "resource_name",
				},
			},
		},
		{
			addr: "resource_type.resource_name[0]",
			want: PartialExpandedResource{
				resource: Resource{
					Mode: ManagedResourceMode,
					Type: "resource_type",
					Name: "resource_name",
				},
			},
			remain: 0,
		},
		{
			addr: "resource_type.resource_name[0].attr",
			want: PartialExpandedResource{
				resource: Resource{
					Mode: ManagedResourceMode,
					Type: "resource_type",
					Name: "resource_name",
				},
			},
			remain: 1,
		},
		{
			addr: "resource.resource_type.resource_name",
			want: PartialExpandedResource{
				resource: Resource{
					Mode: ManagedResourceMode,
					Type: "resource_type",
					Name: "resource_name",
				},
			},
			remain: 0,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.addr, func(t *testing.T) {
			traversal, traversalDiags := hclsyntax.ParseTraversalAbs([]byte(tc.addr), "", hcl.InitialPos)
			if len(traversalDiags) > 0 {
				t.Fatalf("unexpected diagnostics: %v", traversalDiags)
			}

			partial, rest, diags := ParsePartialExpandedResource(traversal)
			if len(diags) > 0 {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}

			if !partial.module.expandedPrefix.Equal(tc.want.module.expandedPrefix) {
				t.Errorf("got expandedPrefix %v; want %v", partial.module.expandedPrefix, tc.want.module.expandedPrefix)
			}
			if !partial.module.unexpandedSuffix.Equal(tc.want.module.unexpandedSuffix) {
				t.Errorf("got unexpandedSuffix %v; want %v", partial.module.unexpandedSuffix, tc.want.module.unexpandedSuffix)
			}
			if !partial.resource.Equal(tc.want.resource) {
				t.Errorf("got resource %v; want %v", partial.resource, tc.want.resource)
			}
			if len(rest) != tc.remain {
				t.Errorf("got %d remaining traversals; want %d", len(rest), tc.remain)
			}
		})
	}
}

func mustParsePartialResourceInstanceStr(s string) AbsResourceInstance {
	r, diags := ParsePartialResourceInstanceStr(s)
	if diags.HasErrors() {
		panic(diags.ErrWithWarnings().Error())
	}
	return r
}
