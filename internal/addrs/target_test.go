// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestContains(t *testing.T) {
	for _, test := range []struct {
		addr, other Targetable
		expect      bool
	}{
		{
			mustParseTargetPattern("module.foo"),
			mustParseTarget("module.bar"),
			false,
		},
		{
			mustParseTargetPattern("module.foo"),
			mustParseTarget("module.foo"),
			true,
		},
		{
			RootModuleInstance,
			mustParseTarget("module.foo"),
			true,
		},
		{
			mustParseTargetPattern("module.foo"),
			RootModuleInstance,
			false,
		},
		{
			mustParseTargetPattern("module.foo"),
			mustParseTarget("module.foo.module.bar[0]"),
			true,
		},
		{
			mustParseTargetPattern("module.foo[2]"),
			mustParseTarget("module.foo[2].module.bar[0]"),
			true,
		},
		{
			mustParseTargetPattern("module.foo"),
			mustParseTarget("module.foo.test_resource.bar"),
			true,
		},
		{
			mustParseTargetPattern("module.foo"),
			mustParseTarget("module.foo.test_resource.bar[0]"),
			true,
		},

		// Resources
		{
			mustParseTargetPattern("test_resource.foo"),
			mustParseTarget("test_resource.foo[\"bar\"]"),
			true,
		},
		{
			mustParseTargetPattern(`test_resource.foo["bar"]`),
			mustParseTarget(`test_resource.foo["bar"]`),
			true,
		},
		{
			mustParseTargetPattern("test_resource.foo"),
			mustParseTarget("test_resource.foo[2]"),
			true,
		},
		{
			mustParseTargetPattern("test_resource.foo"),
			mustParseTarget("module.bar.test_resource.foo[2]"),
			false,
		},
		{
			mustParseTargetPattern("module.bar.test_resource.foo"),
			mustParseTarget("module.bar.test_resource.foo[2]"),
			true,
		},
		{
			// a keyless module step in a target selects every instance
			mustParseTargetPattern("module.bar.test_resource.foo"),
			mustParseTarget("module.bar[0].test_resource.foo[2]"),
			true,
		},
		{
			mustParseTargetPattern("module.bar.test_resource.foo"),
			mustParseTarget("module.bar.test_resource.foo[0]"),
			true,
		},
		{
			mustParseTargetPattern("module.bax"),
			mustParseTarget("module.bax[0].test_resource.foo[0]"),
			true,
		},

		// Config paths, while never returned from parsing a target, must still
		// be targetable
		{
			ConfigResource{
				Module: []string{"bar"},
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "test_resource",
					Name: "foo",
				},
			},
			mustParseTarget("module.bar.test_resource.foo[2]"),
			true,
		},
		{
			mustParseTargetPattern("module.bar"),
			ConfigResource{
				Module: []string{"bar"},
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "test_resource",
					Name: "foo",
				},
			},
			true,
		},
		{
			mustParseTargetPattern("module.bar.test_resource.foo"),
			ConfigResource{
				Module: []string{"bar"},
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "test_resource",
					Name: "foo",
				},
			},
			true,
		},
		{
			ConfigResource{
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "test_resource",
					Name: "foo",
				},
			},
			mustParseTarget("module.bar.test_resource.foo[2]"),
			false,
		},
		{
			ConfigResource{
				Module: []string{"bar"},
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "test_resource",
					Name: "foo",
				},
			},
			mustParseTarget("module.bar[0].test_resource.foo"),
			true,
		},

		// Modules are also never the result of parsing a target, but also need
		// to be targetable
		{
			Module{"bar"},
			Module{"bar", "baz"},
			true,
		},
		{
			Module{"bar"},
			mustParseTarget("module.bar[0]"),
			true,
		},
		{
			// A keyless module step in a target selects every instance, so
			// the pattern contains the entire Module.
			mustParseTargetPattern("module.bar"),
			Module{"bar"},
			true,
		},
		{
			// A specific ModuleInstance cannot contain a module
			mustParseTargetPattern("module.bar[0]"),
			Module{"bar"},
			false,
		},
		{
			Module{"bar", "baz"},
			mustParseTarget("module.bar[0].module.baz.test_resource.foo[1]"),
			true,
		},
		{
			mustParseTargetPattern("module.bar[0].module.baz"),
			Module{"bar", "baz"},
			false,
		},

		// Concrete addresses only contain the exact instances they refer to,
		// so a step without an instance key refers only to the instance of a
		// module call without count or for_each.
		{
			mustParseTarget("module.bar"),
			mustParseTarget("module.bar[0].test_resource.foo"),
			false,
		},
		{
			mustParseTarget("module.bar"),
			mustParseTarget("module.bar.test_resource.foo"),
			true,
		},
		{
			mustParseTarget("module.bar"),
			Module{"bar"},
			false,
		},
		{
			mustParseTarget("module.bar.test_resource.foo"),
			ConfigResource{
				Module: []string{"bar"},
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "test_resource",
					Name: "foo",
				},
			},
			false,
		},
		{
			// a resource instance doesn't contain the whole resource
			RootModuleInstance.ResourceInstance(ManagedResourceMode, "test_resource", "foo", NoKey),
			mustParseTarget("test_resource.foo"),
			false,
		},
	} {
		t.Run(fmt.Sprintf("%s-in-%s", test.other, test.addr), func(t *testing.T) {
			got := test.addr.Contains(test.other)
			if got != test.expect {
				t.Fatalf("expected %q.Contains(%q) == %t", test.addr, test.other, test.expect)
			}
		})
	}
}

func TestContains_patterns(t *testing.T) {
	testResourceX := ConfigResource{
		Module:   Module{"foo"},
		Resource: Resource{Mode: ManagedResourceMode, Type: "test_resource", Name: "x"},
	}

	for _, test := range []struct {
		addr, other Targetable
		expect      bool
	}{
		{
			mustParseTargetPattern("module.foo[*]"),
			mustParseTarget("module.foo[0]"),
			true,
		},
		{
			// a wildcard includes a module call without count or for_each
			mustParseTargetPattern("module.foo[*]"),
			mustParseTarget("module.foo"),
			true,
		},
		{
			mustParseTargetPattern("module.foo[*]"),
			Module{"foo"},
			true,
		},
		{
			mustParseTargetPattern("module.foo[*]"),
			mustParseTarget("module.foo[1].module.bar[0].test_resource.x[2]"),
			true,
		},
		{
			mustParseTargetPattern("module.foo[*]"),
			mustParseTarget("module.bar[0]"),
			false,
		},
		{
			// a specific instance can't contain all instances
			mustParseTarget("module.foo[0]"),
			mustParseTargetPattern("module.foo[*]"),
			false,
		},
		{
			mustParseTargetPattern("module.foo[*].module.bar"),
			mustParseTarget("module.foo[0].module.bar[1]"),
			true,
		},
		{
			mustParseTargetPattern("module.foo[*].module.bar[0]"),
			mustParseTarget("module.foo[1].module.bar[1]"),
			false,
		},
		{
			// a concrete address with a keyless step only contains the
			// unkeyed instance
			mustParseTarget("module.foo.module.bar"),
			mustParseTarget("module.foo[0].module.bar"),
			false,
		},
		{
			// while in a target pattern, any keyless step selects every
			// instance
			mustParseTargetPattern("module.foo.module.bar"),
			mustParseTarget("module.foo[0].module.bar[1]"),
			true,
		},
		{
			mustParseTargetPattern("module.foo.test_resource.x"),
			mustParseTarget("module.foo[0].test_resource.x[1]"),
			true,
		},
		{
			mustParseTargetPattern("module.foo.module.bar[1].test_resource.x[0]"),
			mustParseTarget("module.foo[2].module.bar[1].test_resource.x[0]"),
			true,
		},
		{
			mustParseTargetPattern("module.foo.module.bar[1].test_resource.x[0]"),
			mustParseTarget("module.foo[2].module.bar[0].test_resource.x[0]"),
			false,
		},
		{
			// keyless steps are equivalent to explicit wildcards
			mustParseTargetPattern("module.foo.test_resource.x"),
			mustParseTargetPattern("module.foo[*].test_resource.x[*]"),
			true,
		},
		{
			mustParseTargetPattern("module.foo[*].test_resource.x[*]"),
			mustParseTargetPattern("module.foo.test_resource.x"),
			true,
		},
		{
			mustParseTargetPattern("module.foo"),
			Module{"foo"},
			true,
		},
		{
			Module{"foo"},
			mustParseTargetPattern("module.foo[1].test_resource.x"),
			true,
		},
		{
			// a pattern containing a specific instance can't contain the
			// whole module
			mustParseTargetPattern("module.foo[1]"),
			Module{"foo"},
			false,
		},
		{
			// resource patterns don't contain modules or actions
			mustParseTargetPattern("test_resource.x"),
			RootModuleInstance,
			false,
		},
		{
			mustParseTargetPattern("module.foo"),
			mustParseTargetAction("module.foo[0].action.act.a"),
			true,
		},
		{
			mustParseTargetPattern("module.foo[*].test_resource.x"),
			mustParseTarget("module.foo[0].test_resource.x[1]"),
			true,
		},
		{
			mustParseTargetPattern("module.foo[*].test_resource.x"),
			mustParseTarget("module.foo[0].test_resource.y[1]"),
			false,
		},
		{
			// a resource target doesn't include resources in child modules
			mustParseTargetPattern("module.foo[*].test_resource.x"),
			mustParseTarget("module.foo[0].module.bar.test_resource.x"),
			false,
		},
		{
			mustParseTargetPattern("module.foo[*].test_resource.x"),
			testResourceX,
			true,
		},
		{
			mustParseTarget("module.foo[0].test_resource.x"),
			testResourceX,
			false,
		},
		{
			mustParseTargetPattern("module.foo[*].test_resource.x[0]"),
			mustParseTarget("module.foo[1].test_resource.x[0]"),
			true,
		},
		{
			mustParseTargetPattern("module.foo[*].test_resource.x[0]"),
			mustParseTarget("module.foo[1].test_resource.x[1]"),
			false,
		},
		{
			mustParseTargetPattern("module.foo[*].test_resource.x[0]"),
			testResourceX,
			false,
		},
		{
			mustParseTargetPattern("test_resource.x[*]"),
			mustParseTarget("test_resource.x[0]"),
			true,
		},
		{
			mustParseTargetPattern("test_resource.x[*]"),
			mustParseTarget("test_resource.x"),
			true,
		},
		{
			mustParseTargetPattern("module.foo[0].test_resource.x[*]"),
			mustParseTarget("module.foo[1].test_resource.x[0]"),
			false,
		},
		{
			// wildcard addresses contain matching wildcard addresses
			mustParseTargetPattern("module.foo[*]"),
			mustParseTargetPattern("module.foo[*].test_resource.x"),
			true,
		},
		{
			mustParseTargetPattern("module.foo[*].test_resource.x"),
			mustParseTargetPattern("module.foo[*].test_resource.x[0]"),
			true,
		},
	} {
		t.Run(fmt.Sprintf("%s-in-%s", test.other, test.addr), func(t *testing.T) {
			got := test.addr.Contains(test.other)
			if got != test.expect {
				t.Fatalf("expected %q.Contains(%q) == %t", test.addr, test.other, test.expect)
			}
		})
	}
}

func TestContains_actions(t *testing.T) {
	wildcardFoo := ModuleInstance{{Name: "foo", InstanceKey: WildcardKey}}

	for _, test := range []struct {
		addr, other Targetable
		expect      bool
	}{
		{
			RootModule.Action("act", "a"),
			RootModule.Action("act", "a"),
			true,
		},
		{
			RootModule.Action("act", "a"),
			RootModule.Action("act", "b"),
			false,
		},
		{
			Module{"foo"}.Action("act", "a"),
			mustParseTargetAction("module.foo[0].action.act.a"),
			true,
		},
		{
			Module{"foo"}.Action("act", "a"),
			mustParseTargetAction("module.foo[1].action.act.a[2]"),
			true,
		},
		{
			RootModuleInstance,
			Module{"foo"}.Action("act", "a"),
			true,
		},
		{
			mustParseTargetPattern("module.foo"),
			Module{"foo"}.Action("act", "a"),
			true,
		},
		{
			// a concrete module instance doesn't contain every instance
			mustParseTarget("module.foo"),
			Module{"foo"}.Action("act", "a"),
			false,
		},
		{
			mustParseTarget("module.foo[0]"),
			Module{"foo"}.Action("act", "a"),
			false,
		},
		{
			mustParseTarget("module.foo[0]"),
			mustParseTargetAction("module.foo[0].action.act.a[1]"),
			true,
		},
		{
			// the concrete action only contains the instances in the
			// unkeyed module instance
			mustParseTargetAction("module.foo.action.act.a"),
			Module{"foo"}.Action("act", "a"),
			false,
		},
		{
			// a single module instance can't contain the action from every
			// instance of the module
			mustParseTargetAction("module.foo[0].action.act.a"),
			Module{"foo"}.Action("act", "a"),
			false,
		},
		{
			// a single action instance can't contain every instance
			mustParseTargetAction("action.act.a[1]"),
			RootModule.Action("act", "a"),
			false,
		},
		{
			mustParseTargetAction("action.act.a"),
			mustParseTargetAction("action.act.a[1]"),
			true,
		},
		{
			mustParseTargetAction("module.foo[0].action.act.a"),
			mustParseTargetAction("module.foo[1].action.act.a"),
			false,
		},
		{
			mustParseTargetAction("action.act.a[1]"),
			mustParseTargetAction("action.act.a[1]"),
			true,
		},
		{
			mustParseTargetAction("action.act.a[1]"),
			mustParseTargetAction("action.act.a"),
			false,
		},
		{
			wildcardFoo.Action("act", "a"),
			mustParseTargetAction("module.foo[1].action.act.a[0]"),
			true,
		},
		{
			wildcardFoo.Action("act", "a"),
			Module{"foo"}.Action("act", "a"),
			true,
		},
		{
			RootModuleInstance.ActionInstance("act", "a", WildcardKey),
			mustParseTargetAction("action.act.a[0]"),
			true,
		},
		{
			RootModuleInstance.ActionInstance("act", "a", WildcardKey),
			mustParseTargetAction("action.act.a"),
			true,
		},
		{
			mustParseTargetActionPattern("module.foo.action.act.a"),
			mustParseTargetAction("module.foo[1].action.act.a[0]"),
			true,
		},
		{
			mustParseTargetActionPattern("module.foo.action.act.a"),
			Module{"foo"}.Action("act", "a"),
			true,
		},
		{
			mustParseTargetActionPattern("module.foo[0].action.act.a"),
			Module{"foo"}.Action("act", "a"),
			false,
		},
		{
			mustParseTargetActionPattern("action.act.a[1]"),
			mustParseTargetAction("action.act.a[1]"),
			true,
		},
		{
			mustParseTargetActionPattern("action.act.a[1]"),
			mustParseTargetAction("action.act.a[2]"),
			false,
		},
		{
			Module{"foo"}.Action("act", "a"),
			mustParseTargetActionPattern("module.foo[*].action.act.a[1]"),
			true,
		},
		{
			mustParseTargetActionPattern("action.act.a"),
			mustParseTarget("test_resource.a"),
			false,
		},
		{
			mustParseTargetAction("action.act.a"),
			mustParseTarget("test_resource.a"),
			false,
		},
		{
			RootModule.Resource(ManagedResourceMode, "test_resource", "a"),
			RootModule.Action("act", "a"),
			false,
		},
	} {
		t.Run(fmt.Sprintf("%s-in-%s", test.other, test.addr), func(t *testing.T) {
			got := test.addr.Contains(test.other)
			if got != test.expect {
				t.Fatalf("expected %q.Contains(%q) == %t", test.addr, test.other, test.expect)
			}
		})
	}
}

func TestCouldContain(t *testing.T) {
	testResourceX := ConfigResource{
		Module:   Module{"foo"},
		Resource: Resource{Mode: ManagedResourceMode, Type: "test_resource", Name: "x"},
	}

	for _, test := range []struct {
		addr, other Targetable
		expect      bool
	}{
		// None of the instance keys of a configuration address are known, so
		// it could contain instances selected by more specific targets.
		{
			mustParseTargetPattern("module.foo[0]"),
			testResourceX,
			true,
		},
		{
			mustParseTargetPattern("module.foo[0].test_resource.x[1]"),
			testResourceX,
			true,
		},
		{
			mustParseTarget("module.foo[0].test_resource.x[1]"),
			testResourceX,
			true,
		},
		{
			mustParseTarget("module.foo[0]"),
			testResourceX,
			true,
		},
		{
			mustParseTargetPattern("module.foo[0]"),
			Module{"foo", "bar"},
			true,
		},
		{
			mustParseTargetPattern("module.bar[0]"),
			testResourceX,
			false,
		},
		{
			mustParseTargetPattern("module.foo[0].test_resource.y"),
			testResourceX,
			false,
		},
		{
			// a resource in the root module is not in a child module
			mustParseTargetPattern("test_resource.x"),
			testResourceX,
			false,
		},
		{
			mustParseTargetActionPattern("module.foo[0].action.test_resource.x"),
			testResourceX,
			false,
		},
		{
			mustParseTargetActionPattern("module.foo[0].action.act.a[1]"),
			Module{"foo"}.Action("act", "a"),
			true,
		},

		// Wildcards in the other address are instance keys which are not
		// known yet, and so could be any instance.
		{
			mustParseTargetPattern("module.foo[0].test_resource.x[1]"),
			mustParseTargetPattern("module.foo[*].test_resource.x[*]"),
			true,
		},
		{
			mustParseTargetPattern("module.foo[0].module.bar[1]"),
			mustParseTargetPattern("module.foo[*].module.bar[*]"),
			true,
		},
		{
			// known instance keys must still match
			mustParseTargetPattern("module.foo[0].module.bar[1]"),
			mustParseTargetPattern("module.foo[1].module.bar[*]"),
			false,
		},

		// Without any wildcards in the other address, this is the same as
		// Contains.
		{
			mustParseTargetPattern("module.foo"),
			mustParseTarget("module.foo[1].test_resource.x[0]"),
			true,
		},
		{
			mustParseTargetPattern("module.foo[0]"),
			mustParseTarget("module.foo[1].test_resource.x[0]"),
			false,
		},
		{
			// a concrete address with a keyless step only contains the
			// unkeyed instance
			mustParseTarget("module.foo.test_resource.x"),
			mustParseTarget("module.foo[0].test_resource.x"),
			false,
		},
	} {
		t.Run(fmt.Sprintf("%s-in-%s", test.other, test.addr), func(t *testing.T) {
			got := CouldContain(test.addr, test.other)
			if got != test.expect {
				t.Fatalf("expected CouldContain(%q, %q) == %t", test.addr, test.other, test.expect)
			}
			if !got && test.addr.Contains(test.other) {
				t.Fatalf("%q contains %q, so CouldContain must also be true", test.addr, test.other)
			}
		})
	}
}

// mustParseTarget parses the concrete address of a module instance, resource,
// or resource instance.
func mustParseTarget(str string) Targetable {
	addr, diags := ParseAbsTargetableStr(str)
	if diags.HasErrors() {
		panic(fmt.Sprintf("%s: %s", str, diags.ErrWithWarnings()))
	}
	return addr
}

// mustParseTargetPattern parses a target in the same way as the -target
// option.
func mustParseTargetPattern(str string) Targetable {
	pattern, diags := ParseTargetStr(str)
	if diags.HasErrors() {
		panic(fmt.Sprintf("%s: %s", str, diags.ErrWithWarnings()))
	}
	return pattern
}

// mustParseTargetActionPattern parses a target in the same way as the -invoke
// option.
func mustParseTargetActionPattern(str string) Targetable {
	pattern, diags := ParseTargetActionStr(str)
	if diags.HasErrors() {
		panic(fmt.Sprintf("%s: %s", str, diags.ErrWithWarnings()))
	}
	return pattern
}

// mustParseTargetAction parses the concrete address of an action or action
// instance.
func mustParseTargetAction(str string) Targetable {
	traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.InitialPos)
	if hclDiags.HasErrors() {
		panic(fmt.Sprintf("%s: %s", str, hclDiags.Error()))
	}
	addr, diags := parseAbsActionTarget(traversal)
	if diags.HasErrors() {
		panic(fmt.Sprintf("%s: %s", str, diags.ErrWithWarnings()))
	}
	return addr
}
