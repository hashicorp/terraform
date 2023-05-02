// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import (
	"fmt"
	"testing"
)

func TestTargetContains(t *testing.T) {
	for _, test := range []struct {
		addr, other Targetable
		expect      bool
	}{
		{
			mustParseTarget("module.foo"),
			mustParseTarget("module.bar"),
			false,
		},
		{
			mustParseTarget("module.foo"),
			mustParseTarget("module.foo"),
			true,
		},
		{
			RootModuleInstance,
			mustParseTarget("module.foo"),
			true,
		},
		{
			mustParseTarget("module.foo"),
			RootModuleInstance,
			false,
		},
		{
			mustParseTarget("module.foo"),
			mustParseTarget("module.foo.module.bar[0]"),
			true,
		},
		{
			mustParseTarget("module.foo"),
			mustParseTarget("module.foo.module.bar[0]"),
			true,
		},
		{
			mustParseTarget("module.foo[2]"),
			mustParseTarget("module.foo[2].module.bar[0]"),
			true,
		},
		{
			mustParseTarget("module.foo"),
			mustParseTarget("module.foo.test_resource.bar"),
			true,
		},
		{
			mustParseTarget("module.foo"),
			mustParseTarget("module.foo.test_resource.bar[0]"),
			true,
		},

		// Resources
		{
			mustParseTarget("test_resource.foo"),
			mustParseTarget("test_resource.foo[\"bar\"]"),
			true,
		},
		{
			mustParseTarget(`test_resource.foo["bar"]`),
			mustParseTarget(`test_resource.foo["bar"]`),
			true,
		},
		{
			mustParseTarget("test_resource.foo"),
			mustParseTarget("test_resource.foo[2]"),
			true,
		},
		{
			mustParseTarget("test_resource.foo"),
			mustParseTarget("module.bar.test_resource.foo[2]"),
			false,
		},
		{
			mustParseTarget("module.bar.test_resource.foo"),
			mustParseTarget("module.bar.test_resource.foo[2]"),
			true,
		},
		{
			mustParseTarget("module.bar.test_resource.foo"),
			mustParseTarget("module.bar[0].test_resource.foo[2]"),
			false,
		},
		{
			mustParseTarget("module.bar.test_resource.foo"),
			mustParseTarget("module.bar.test_resource.foo[0]"),
			true,
		},
		{
			mustParseTarget("module.bax"),
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
			mustParseTarget("module.bar"),
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
			mustParseTarget("module.bar.test_resource.foo"),
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
			// Parsing an ambiguous module path needs to ensure the
			// ModuleInstance could contain the Module. This is safe because if
			// the module could be expanded, it must have an index, meaning no
			// index indicates that the module instance and module are
			// functionally equivalent.
			mustParseTarget("module.bar"),
			Module{"bar"},
			true,
		},
		{
			// A specific ModuleInstance cannot contain a module
			mustParseTarget("module.bar[0]"),
			Module{"bar"},
			false,
		},
		{
			Module{"bar", "baz"},
			mustParseTarget("module.bar[0].module.baz.test_resource.foo[1]"),
			true,
		},
		{
			mustParseTarget("module.bar[0].module.baz"),
			Module{"bar", "baz"},
			false,
		},
	} {
		t.Run(fmt.Sprintf("%s-in-%s", test.other, test.addr), func(t *testing.T) {
			got := test.addr.TargetContains(test.other)
			if got != test.expect {
				t.Fatalf("expected %q.TargetContains(%q) == %t", test.addr, test.other, test.expect)
			}
		})
	}
}

func TestResourceContains(t *testing.T) {
	for _, test := range []struct {
		in, other Targetable
		expect    bool
	}{} {
		t.Run(fmt.Sprintf("%s-in-%s", test.other, test.in), func(t *testing.T) {
			got := test.in.TargetContains(test.other)
			if got != test.expect {
				t.Fatalf("expected %q.TargetContains(%q) == %t", test.in, test.other, test.expect)
			}
		})
	}
}

func mustParseTarget(str string) Targetable {
	t, diags := ParseTargetStr(str)
	if diags != nil {
		panic(fmt.Sprintf("%s: %s", str, diags.ErrWithWarnings()))
	}
	return t.Subject
}
