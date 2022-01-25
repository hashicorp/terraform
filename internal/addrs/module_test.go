package addrs

import (
	"fmt"
	"testing"
)

func TestModuleEqual_true(t *testing.T) {
	modules := []Module{
		RootModule,
		{"a"},
		{"a", "b"},
		{"a", "b", "c"},
	}
	for _, m := range modules {
		t.Run(m.String(), func(t *testing.T) {
			if !m.Equal(m) {
				t.Fatalf("expected %#v to be equal to itself", m)
			}
		})
	}
}

func TestModuleEqual_false(t *testing.T) {
	testCases := []struct {
		left  Module
		right Module
	}{
		{
			RootModule,
			Module{"a"},
		},
		{
			Module{"a"},
			Module{"b"},
		},
		{
			Module{"a"},
			Module{"a", "a"},
		},
		{
			Module{"a", "b"},
			Module{"a", "B"},
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
