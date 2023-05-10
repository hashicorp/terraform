// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import (
	"fmt"
	"testing"
)

func TestModuleInstanceEqual_true(t *testing.T) {
	addrs := []string{
		"module.foo",
		"module.foo.module.bar",
		"module.foo[1].module.bar",
		`module.foo["a"].module.bar["b"]`,
		`module.foo["a"].module.bar.module.baz[3]`,
	}
	for _, m := range addrs {
		t.Run(m, func(t *testing.T) {
			addr, diags := ParseModuleInstanceStr(m)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %s", diags.Err())
			}
			if !addr.Equal(addr) {
				t.Fatalf("expected %#v to be equal to itself", addr)
			}
		})
	}
}

func TestModuleInstanceEqual_false(t *testing.T) {
	testCases := []struct {
		left  string
		right string
	}{
		{
			"module.foo",
			"module.bar",
		},
		{
			"module.foo",
			"module.foo.module.bar",
		},
		{
			"module.foo[1]",
			"module.bar[1]",
		},
		{
			`module.foo[1]`,
			`module.foo["1"]`,
		},
		{
			"module.foo.module.bar",
			"module.foo[1].module.bar",
		},
		{
			`module.foo.module.bar`,
			`module.foo["a"].module.bar`,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s = %s", tc.left, tc.right), func(t *testing.T) {
			left, diags := ParseModuleInstanceStr(tc.left)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags parsing %s: %s", tc.left, diags.Err())
			}
			right, diags := ParseModuleInstanceStr(tc.right)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags parsing %s: %s", tc.right, diags.Err())
			}

			if left.Equal(right) {
				t.Fatalf("expected %#v not to be equal to %#v", left, right)
			}

			if right.Equal(left) {
				t.Fatalf("expected %#v not to be equal to %#v", right, left)
			}
		})
	}
}

func BenchmarkStringShort(b *testing.B) {
	addr, _ := ParseModuleInstanceStr(`module.foo`)
	for n := 0; n < b.N; n++ {
		addr.String()
	}
}

func BenchmarkStringLong(b *testing.B) {
	addr, _ := ParseModuleInstanceStr(`module.southamerica-brazil-region.module.user-regional-desktops.module.user-name`)
	for n := 0; n < b.N; n++ {
		addr.String()
	}
}

func TestModuleInstance_IsDeclaredByCall(t *testing.T) {
	tests := []struct {
		instance ModuleInstance
		call     AbsModuleCall
		want     bool
	}{
		{
			ModuleInstance{},
			AbsModuleCall{},
			false,
		},
		{
			mustParseModuleInstanceStr("module.child"),
			AbsModuleCall{},
			false,
		},
		{
			ModuleInstance{},
			AbsModuleCall{
				RootModuleInstance,
				ModuleCall{Name: "child"},
			},
			false,
		},
		{
			mustParseModuleInstanceStr("module.child"),
			AbsModuleCall{ // module.child
				RootModuleInstance,
				ModuleCall{Name: "child"},
			},
			true,
		},
		{
			mustParseModuleInstanceStr(`module.child`),
			AbsModuleCall{ // module.kinder.module.child
				mustParseModuleInstanceStr("module.kinder"),
				ModuleCall{Name: "child"},
			},
			false,
		},
		{
			mustParseModuleInstanceStr("module.kinder"),
			// module.kinder.module.child contains module.kinder, but is not itself an instance of module.kinder
			AbsModuleCall{
				mustParseModuleInstanceStr("module.kinder"),
				ModuleCall{Name: "child"},
			},
			false,
		},
		{
			mustParseModuleInstanceStr("module.child"),
			AbsModuleCall{
				mustParseModuleInstanceStr(`module.kinder["a"]`),
				ModuleCall{Name: "kinder"},
			},
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%q.IsCallInstance(%q)", test.instance, test.call.String()), func(t *testing.T) {
			got := test.instance.IsDeclaredByCall(test.call)
			if got != test.want {
				t.Fatal("wrong result")
			}
		})
	}
}

func mustParseModuleInstanceStr(str string) ModuleInstance {
	mi, diags := ParseModuleInstanceStr(str)
	if diags.HasErrors() {
		panic(diags.ErrWithWarnings())
	}
	return mi
}
