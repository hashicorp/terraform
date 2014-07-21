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

func TestNewResourceVariable(t *testing.T) {
	v, err := NewResourceVariable("foo.bar.baz")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if v.Type != "foo" {
		t.Fatalf("bad: %#v", v)
	}
	if v.Name != "bar" {
		t.Fatalf("bad: %#v", v)
	}
	if v.Field != "baz" {
		t.Fatalf("bad: %#v", v)
	}
	if v.Multi {
		t.Fatal("should not be multi")
	}

	if v.FullKey() != "foo.bar.baz" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestNewUserVariable(t *testing.T) {
	v, err := NewUserVariable("var.bar")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if v.Name != "bar" {
		t.Fatalf("bad: %#v", v.Name)
	}
	if v.FullKey() != "var.bar" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestReplaceVariables(t *testing.T) {
	input := "foo-${var.bar}"
	expected := "foo-bar"

	unk, err := ReplaceVariables(&input, map[string]string{
		"var.bar": "bar",
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if len(unk) > 0 {
		t.Fatal("bad: %#v", unk)
	}

	if input != expected {
		t.Fatalf("bad: %#v", input)
	}
}

func TestResourceVariable_impl(t *testing.T) {
	var _ InterpolatedVariable = new(ResourceVariable)
}

func TestResourceVariable_Multi(t *testing.T) {
	v, err := NewResourceVariable("foo.bar.*.baz")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if v.Type != "foo" {
		t.Fatalf("bad: %#v", v)
	}
	if v.Name != "bar" {
		t.Fatalf("bad: %#v", v)
	}
	if v.Field != "baz" {
		t.Fatalf("bad: %#v", v)
	}
	if !v.Multi {
		t.Fatal("should be multi")
	}
}

func TestResourceVariable_MultiIndex(t *testing.T) {
	cases := []struct {
		Input string
		Index int
		Field string
	}{
		{"foo.bar.*.baz", -1, "baz"},
		{"foo.bar.0.baz", 0, "baz"},
		{"foo.bar.5.baz", 5, "baz"},
	}

	for _, tc := range cases {
		v, err := NewResourceVariable(tc.Input)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		if !v.Multi {
			t.Fatalf("should be multi: %s", tc.Input)
		}
		if v.Index != tc.Index {
			t.Fatalf("bad: %d\n\n%s", v.Index, tc.Input)
		}
		if v.Field != tc.Field {
			t.Fatalf("bad: %s\n\n%s", v.Field, tc.Input)
		}
	}
}

func TestUserVariable_impl(t *testing.T) {
	var _ InterpolatedVariable = new(UserVariable)
}

func TestUserMapVariable_impl(t *testing.T) {
	var _ InterpolatedVariable = new(UserMapVariable)
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

func TestVariableDetectWalker_resourceMulti(t *testing.T) {
	w := new(variableDetectWalker)

	str := `foo ${ec2.foo.*.bar}`
	if err := w.Primitive(reflect.ValueOf(str)); err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(w.Variables) != 1 {
		t.Fatalf("bad: %#v", w.Variables)
	}
	if w.Variables["ec2.foo.*.bar"].(*ResourceVariable).FullKey() != "ec2.foo.*.bar" {
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

func TestVariableDetectWalker_userMap(t *testing.T) {
	w := new(variableDetectWalker)

	str := `foo ${var.foo.bar}`
	if err := w.Primitive(reflect.ValueOf(str)); err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(w.Variables) != 1 {
		t.Fatalf("bad: %#v", w.Variables)
	}

	v := w.Variables["var.foo.bar"].(*UserMapVariable)
	if v.FullKey() != "var.foo.bar" {
		t.Fatalf("bad: %#v", w.Variables)
	}
	if v.Name != "foo" {
		t.Fatalf("bad: %#v", w.Variables)
	}
	if v.Elem != "bar" {
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

func TestVariableReplaceWalker_unknown(t *testing.T) {
	cases := []struct {
		Input  interface{}
		Output interface{}
		Keys   []string
	}{
		{
			map[string]interface{}{
				"foo": "bar",
				"bar": "hello${var.unknown}world",
			},
			map[string]interface{}{
				"foo": "bar",
			},
			[]string{"bar"},
		},
		{
			map[string]interface{}{
				"foo": []string{"foo", "${var.unknown}", "bar"},
			},
			map[string]interface{}{},
			[]string{"foo"},
		},
		{
			map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "${var.unknown}",
				},
			},
			map[string]interface{}{
				"foo": map[string]interface{}{},
			},
			[]string{"foo.bar"},
		},
	}

	for i, tc := range cases {
		var input interface{} = tc.Input
		w := &variableReplaceWalker{
			Values: map[string]string{
				"var.unknown": UnknownVariableValue,
			},
		}

		if reflect.ValueOf(tc.Input).Kind() == reflect.String {
			input = &tc.Input
		}

		if err := reflectwalk.Walk(input, w); err != nil {
			t.Fatalf("err: %s", err)
		}

		if !reflect.DeepEqual(tc.Input, tc.Output) {
			t.Fatalf("bad %d: %#v", i, tc.Input)
		}

		if !reflect.DeepEqual(tc.Keys, w.UnknownKeys) {
			t.Fatalf("bad: %#v", w.UnknownKeys)
		}
	}
}
