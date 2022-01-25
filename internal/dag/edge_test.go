package dag

import (
	"testing"
)

func TestBasicEdgeHashcode(t *testing.T) {
	e1 := BasicEdge(1, 2)
	e2 := BasicEdge(1, 2)
	if e1.Hashcode() != e2.Hashcode() {
		t.Fatalf("bad")
	}
}

func TestBasicEdgeHashcode_pointer(t *testing.T) {
	type test struct {
		Value string
	}

	v1, v2 := &test{"foo"}, &test{"bar"}
	e1 := BasicEdge(v1, v2)
	e2 := BasicEdge(v1, v2)
	if e1.Hashcode() != e2.Hashcode() {
		t.Fatalf("bad")
	}
}
