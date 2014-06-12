package config

import (
	"reflect"
	"testing"

	"github.com/mitchellh/reflectwalk"
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

func BenchmarkVariableReplaceWalker(b *testing.B) {
	w := &variableReplaceWalker{
		Values: map[string]string{
			"var.bar":      "bar",
			"bar.baz.bing": "baz",
		},
	}

	str := `foo ${var.bar} bar ${bar.baz.bing} $${escaped}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := reflectwalk.Walk(&str, w); err != nil {
			panic(err)
		}
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

func TestVariableDetectWalker_resource(t *testing.T) {
	w := new(variableDetectWalker)

	str := `foo ${ec2.foo.bar}`
	if err := w.Primitive(reflect.ValueOf(str)); err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(w.Variables) != 1 {
		t.Fatalf("bad: %#v", w.Variables)
	}
	if w.Variables["ec2.foo.bar"].(*ResourceVariable).FullKey() != "ec2.foo.bar" {
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

func TestVariableReplaceWalker(t *testing.T) {
	w := &variableReplaceWalker{
		Values: map[string]string{
			"var.bar":     "bar",
			"var.unknown": UnknownVariableValue,
		},
	}

	cases := []struct {
		Input  interface{}
		Output interface{}
	}{
		{
			`foo ${var.bar}`,
			"foo bar",
		},
		{
			[]string{"foo", "${var.bar}"},
			[]string{"foo", "bar"},
		},
		{
			map[string]interface{}{
				"ami": "${var.bar}",
				"security_groups": []interface{}{
					"foo",
					"${var.bar}",
				},
			},
			map[string]interface{}{
				"ami": "bar",
				"security_groups": []interface{}{
					"foo",
					"bar",
				},
			},
		},
		{
			map[string]interface{}{
				"foo": map[string]interface{}{
					"foo": []string{"${var.bar}"},
				},
			},
			map[string]interface{}{
				"foo": map[string]interface{}{
					"foo": []string{"bar"},
				},
			},
		},
		{
			map[string]interface{}{
				"foo": "bar",
				"bar": "hello${var.unknown}world",
			},
			map[string]interface{}{
				"foo": "bar",
			},
		},
		{
			map[string]interface{}{
				"foo": []string{"foo", "${var.unknown}", "bar"},
			},
			map[string]interface{}{},
		},
	}

	for i, tc := range cases {
		var input interface{} = tc.Input
		if reflect.ValueOf(tc.Input).Kind() == reflect.String {
			input = &tc.Input
		}

		if err := reflectwalk.Walk(input, w); err != nil {
			t.Fatalf("err: %s", err)
		}

		if !reflect.DeepEqual(tc.Input, tc.Output) {
			t.Fatalf("bad %d: %#v", i, tc.Input)
		}
	}
}
