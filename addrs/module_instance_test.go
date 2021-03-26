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
