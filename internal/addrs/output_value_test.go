package addrs

import (
	"fmt"
	"testing"
)

func TestAbsOutputValueInstanceEqual_true(t *testing.T) {
	foo, diags := ParseModuleInstanceStr("module.foo")
	if len(diags) > 0 {
		t.Fatalf("unexpected diags: %s", diags.Err())
	}
	foobar, diags := ParseModuleInstanceStr("module.foo[1].module.bar")
	if len(diags) > 0 {
		t.Fatalf("unexpected diags: %s", diags.Err())
	}

	ovs := []AbsOutputValue{
		foo.OutputValue("a"),
		foobar.OutputValue("b"),
	}
	for _, r := range ovs {
		t.Run(r.String(), func(t *testing.T) {
			if !r.Equal(r) {
				t.Fatalf("expected %#v to be equal to itself", r)
			}
		})
	}
}

func TestAbsOutputValueInstanceEqual_false(t *testing.T) {
	foo, diags := ParseModuleInstanceStr("module.foo")
	if len(diags) > 0 {
		t.Fatalf("unexpected diags: %s", diags.Err())
	}
	foobar, diags := ParseModuleInstanceStr("module.foo[1].module.bar")
	if len(diags) > 0 {
		t.Fatalf("unexpected diags: %s", diags.Err())
	}

	testCases := []struct {
		left  AbsOutputValue
		right AbsOutputValue
	}{
		{
			foo.OutputValue("a"),
			foo.OutputValue("b"),
		},
		{
			foo.OutputValue("a"),
			foobar.OutputValue("a"),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s = %s", tc.left, tc.right), func(t *testing.T) {
			if tc.left.Equal(tc.right) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.left, tc.right)
			}

			if tc.right.Equal(tc.left) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.right, tc.left)
			}
		})
	}
}
