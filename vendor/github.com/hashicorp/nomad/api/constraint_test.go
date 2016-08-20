package api

import (
	"reflect"
	"testing"
)

func TestCompose_Constraints(t *testing.T) {
	c := NewConstraint("kernel.name", "=", "darwin")
	expect := &Constraint{
		LTarget: "kernel.name",
		RTarget: "darwin",
		Operand: "=",
	}
	if !reflect.DeepEqual(c, expect) {
		t.Fatalf("expect: %#v, got: %#v", expect, c)
	}
}
