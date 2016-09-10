package terraform

import (
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
			// Resource name without enough parts is invalid
			ResourceName: "aws",
			Alias:        "",
			Expected:     "", // intentionally not a valid provider name
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
