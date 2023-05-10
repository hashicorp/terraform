// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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

func TestModuleString(t *testing.T) {
	testCases := map[string]Module{
		"": {},
		"module.alpha": {
			"alpha",
		},
		"module.alpha.module.beta": {
			"alpha",
			"beta",
		},
		"module.alpha.module.beta.module.charlie": {
			"alpha",
			"beta",
			"charlie",
		},
	}
	for str, module := range testCases {
		t.Run(str, func(t *testing.T) {
			if got, want := module.String(), str; got != want {
				t.Errorf("wrong result: got %q, want %q", got, want)
			}
		})
	}
}

func BenchmarkModuleStringShort(b *testing.B) {
	module := Module{"a", "b"}
	for n := 0; n < b.N; n++ {
		module.String()
	}
}

func BenchmarkModuleStringLong(b *testing.B) {
	module := Module{"southamerica-brazil-region", "user-regional-desktop", "user-name"}
	for n := 0; n < b.N; n++ {
		module.String()
	}
}
