package terraform

import "testing"

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
