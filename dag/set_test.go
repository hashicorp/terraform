package dag

import (
	"fmt"
	"testing"
)

func TestSetDifference(t *testing.T) {
	cases := []struct {
		Name     string
		A, B     []interface{}
		Expected []interface{}
	}{
		{
			"same",
			[]interface{}{1, 2, 3},
			[]interface{}{3, 1, 2},
			[]interface{}{},
		},

		{
			"A has extra elements",
			[]interface{}{1, 2, 3},
			[]interface{}{3, 2},
			[]interface{}{1},
		},

		{
			"B has extra elements",
			[]interface{}{1, 2, 3},
			[]interface{}{3, 2, 1, 4},
			[]interface{}{},
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
		Input    []interface{}
		Expected []interface{}
	}{
		{
			[]interface{}{1, 2, 3},
			[]interface{}{1, 2, 3},
		},

		{
			[]interface{}{4, 5, 6},
			[]interface{}{4},
		},

		{
			[]interface{}{7, 8, 9},
			[]interface{}{},
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

			actual := input.Filter(func(v interface{}) bool {
				return v.(int) < 5
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
	a.Add(1)
	a.Add(2)

	b := a.Copy()
	b.Add(3)

	diff := b.Difference(a)

	if diff.Len() != 1 {
		t.Fatalf("expected single diff value, got %#v", diff)
	}

	if !diff.Include(3) {
		t.Fatalf("diff does not contain 3, got %#v", diff)
	}

}

func BenchmarkSetIntersectionLargeToSmall(b *testing.B) {
	var small, large Set
	for i := 0; i < b.N; i++ {
		large.Add(i)
	}
	small.Add(1)
	b.ResetTimer()
	large.Intersection(small)
}
