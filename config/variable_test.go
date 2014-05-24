package config

import (
	"reflect"
	"testing"
)

func BenchmarkVariableDetectWalker(b *testing.B) {
	w := new(variableDetectWalker)
	str := reflect.ValueOf(`foo ${var.bar} bar ${bar.baz.bing} $${escaped}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Variables = nil
		w.Primitive(str)
	}
}

func TestVariableDetectWalker(t *testing.T) {
	w := new(variableDetectWalker)

	str := `foo ${var.bar}`
	if err := w.Primitive(reflect.ValueOf(str)); err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(w.Variables) != 1 {
		t.Fatalf("bad: %#v", w.Variables)
	}
	if w.Variables["var.bar"].(*UserVariable).FullKey() != "var.bar" {
		t.Fatalf("bad: %#v", w.Variables)
	}
}

func TestVariableDetectWalker_bad(t *testing.T) {
	w := new(variableDetectWalker)

	str := `foo ${bar}`
	if err := w.Primitive(reflect.ValueOf(str)); err == nil {
		t.Fatal("should error")
	}
}

func TestVariableDetectWalker_escaped(t *testing.T) {
	w := new(variableDetectWalker)

	str := `foo $${var.bar}`
	if err := w.Primitive(reflect.ValueOf(str)); err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(w.Variables) > 0 {
		t.Fatalf("bad: %#v", w.Variables)
	}
}

func TestVariableDetectWalker_empty(t *testing.T) {
	w := new(variableDetectWalker)

	str := `foo`
	if err := w.Primitive(reflect.ValueOf(str)); err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(w.Variables) > 0 {
		t.Fatalf("bad: %#v", w.Variables)
	}
}

