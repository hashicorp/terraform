// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"fmt"
	"testing"
)

func TestSetDifference(t *testing.T) {
	cases := []struct {
		Name     string
		A, B     []testV
		Expected []testV
	}{
		{
			"same",
			[]testV{1, 2, 3},
			[]testV{3, 1, 2},
			[]testV{},
		},

		{
			"A has extra elements",
			[]testV{1, 2, 3},
			[]testV{3, 2},
			[]testV{1},
		},

		{
			"B has extra elements",
			[]testV{1, 2, 3},
			[]testV{3, 2, 1, 4},
			[]testV{},
		},
		{
			"B is nil",
			[]testV{1, 2, 3},
			nil,
			[]testV{1, 2, 3},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			one := make(Set)
			two := make(Set)
			expected := make(Set)
			for _, v := range tc.A {
				one.Add(v)
			}
			for _, v := range tc.B {
				two.Add(v)
			}
			if tc.B == nil {
				two = nil
			}
			for _, v := range tc.Expected {
				expected.Add(v)
			}

			actual := one.Difference(two)
			match := actual.Intersection(expected)
			if match.Len() != expected.Len() {
				t.Fatalf("bad: %#v", actual.List())
			}
		})
	}
}

func TestSetFilter(t *testing.T) {
	cases := []struct {
		Input    []testV
		Expected []testV
	}{
		{
			[]testV{1, 2, 3},
			[]testV{1, 2, 3},
		},

		{
			[]testV{4, 5, 6},
			[]testV{4},
		},

		{
			[]testV{7, 8, 9},
			[]testV{},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%#v", i, tc.Input), func(t *testing.T) {
			input := make(Set)
			expected := make(Set)
			for _, v := range tc.Input {
				input.Add(v)
			}
			for _, v := range tc.Expected {
				expected.Add(v)
			}

			actual := input.Filter(func(v Vertex) bool {
				return v.(testV) < 5
			})
			match := actual.Intersection(expected)
			if match.Len() != expected.Len() {
				t.Fatalf("bad: %#v", actual.List())
			}
		})
	}
}

func TestSetCopy(t *testing.T) {
	a := make(Set)
	a.Add(testV(1))
	a.Add(testV(2))

	b := a.Copy()
	b.Add(testV(3))

	diff := b.Difference(a)

	if diff.Len() != 1 {
		t.Fatalf("expected single diff value, got %#v", diff)
	}

	if !diff.Include(testV(3)) {
		t.Fatalf("diff does not contain 3, got %#v", diff)
	}

}

func makeSet(n int) Set {
	ret := make(Set, n)
	for i := range n {
		ret.Add(testV(i))
	}
	return ret
}

func BenchmarkSetIntersection_100_100000(b *testing.B) {
	small := makeSet(100)
	large := makeSet(100000)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		small.Intersection(large)
	}
}

func BenchmarkSetIntersection_100000_100(b *testing.B) {
	small := makeSet(100)
	large := makeSet(100000)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		large.Intersection(small)
	}
}
