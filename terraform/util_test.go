package terraform

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestSemaphore(t *testing.T) {
	s := NewSemaphore(2)
	timer := time.AfterFunc(time.Second, func() {
		panic("deadlock")
	})
	defer timer.Stop()

	s.Acquire()
	if !s.TryAcquire() {
		t.Fatalf("should acquire")
	}
	if s.TryAcquire() {
		t.Fatalf("should not acquire")
	}
	s.Release()
	s.Release()

	// This release should panic
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("should panic")
		}
	}()
	s.Release()
}

func TestStrSliceContains(t *testing.T) {
	if strSliceContains(nil, "foo") {
		t.Fatalf("Bad")
	}
	if strSliceContains([]string{}, "foo") {
		t.Fatalf("Bad")
	}
	if strSliceContains([]string{"bar"}, "foo") {
		t.Fatalf("Bad")
	}
	if !strSliceContains([]string{"bar", "foo"}, "foo") {
		t.Fatalf("Bad")
	}
}

func TestUtilResourceProvider(t *testing.T) {
	type testCase struct {
		ResourceName string
		Alias        string
		Expected     string
	}

	tests := []testCase{
		{
			// If no alias is provided, the first underscore-separated segment
			// is assumed to be the provider name.
			ResourceName: "aws_thing",
			Alias:        "",
			Expected:     "aws",
		},
		{
			// If we have more than one underscore then it's the first one that we'll use.
			ResourceName: "aws_thingy_thing",
			Alias:        "",
			Expected:     "aws",
		},
		{
			// A provider can export a resource whose name is just the bare provider name,
			// e.g. because the provider only has one resource and so any additional
			// parts would be redundant.
			ResourceName: "external",
			Alias:        "",
			Expected:     "external",
		},
		{
			// Alias always overrides the default extraction of the name
			ResourceName: "aws_thing",
			Alias:        "tls.baz",
			Expected:     "tls.baz",
		},
	}

	for _, test := range tests {
		got := resourceProvider(test.ResourceName, test.Alias)
		if got != test.Expected {
			t.Errorf(
				"(%q, %q) produced %q; want %q",
				test.ResourceName, test.Alias,
				got,
				test.Expected,
			)
		}
	}
}

func TestUniqueStrings(t *testing.T) {
	cases := []struct {
		Input    []string
		Expected []string
	}{
		{
			[]string{},
			[]string{},
		},
		{
			[]string{"x"},
			[]string{"x"},
		},
		{
			[]string{"a", "b", "c"},
			[]string{"a", "b", "c"},
		},
		{
			[]string{"a", "a", "a"},
			[]string{"a"},
		},
		{
			[]string{"a", "b", "a", "b", "a", "a"},
			[]string{"a", "b"},
		},
		{
			[]string{"c", "b", "a", "c", "b"},
			[]string{"a", "b", "c"},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("unique-%d", i), func(t *testing.T) {
			actual := uniqueStrings(tc.Input)
			if !reflect.DeepEqual(tc.Expected, actual) {
				t.Fatalf("Expected: %q\nGot: %q", tc.Expected, actual)
			}
		})
	}
}
