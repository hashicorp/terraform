package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestNewInterpolation(t *testing.T) {
	cases := []struct {
		Input  string
		Result Interpolation
		Error  bool
	}{
		{
			"foo",
			nil,
			true,
		},

		{
			"var.foo",
			&VariableInterpolation{
				Variable: &UserVariable{
					Name: "foo",
					key:  "var.foo",
				},
				key: "var.foo",
			},
			false,
		},

		{
			"lookup(var.foo, var.bar)",
			&FunctionInterpolation{
				Func: nil, // Funcs["lookup"]
				Args: []InterpolatedVariable{
					&UserVariable{
						Name: "foo",
						key:  "var.foo",
					},
					&UserVariable{
						Name: "bar",
						key:  "var.bar",
					},
				},
				key: "lookup(var.foo, var.bar)",
			},
			false,
		},
	}

	for i, tc := range cases {
		actual, err := NewInterpolation(tc.Input)
		if (err != nil) != tc.Error {
			t.Fatalf("%d. Error: %s", i, err)
		}

		// This is jank, but reflect.DeepEqual never has functions
		// being the same.
		if f, ok := actual.(*FunctionInterpolation); ok {
			f.Func = nil
		}

		if !reflect.DeepEqual(actual, tc.Result) {
			t.Fatalf("%d bad: %#v", i, actual)
		}
	}
}

func TestNewInterpolatedVariable(t *testing.T) {
	cases := []struct {
		Input  string
		Result InterpolatedVariable
		Error  bool
	}{
		{
			"var.foo",
			&UserVariable{
				Name: "foo",
				key:  "var.foo",
			},
			false,
		},
	}

	for i, tc := range cases {
		actual, err := NewInterpolatedVariable(tc.Input)
		if (err != nil) != tc.Error {
			t.Fatalf("%d. Error: %s", i, err)
		}
		if !reflect.DeepEqual(actual, tc.Result) {
			t.Fatalf("%d bad: %#v", i, actual)
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

func TestFunctionInterpolation_impl(t *testing.T) {
	var _ Interpolation = new(FunctionInterpolation)
}

func TestFunctionInterpolation(t *testing.T) {
	v1, err := NewInterpolatedVariable("var.foo")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	v2, err := NewInterpolatedVariable("var.bar")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	fn := func(vs map[string]string, args ...string) (string, error) {
		return strings.Join(args, " "), nil
	}

	i := &FunctionInterpolation{
		Func: fn,
		Args: []InterpolatedVariable{v1, v2},
		key:  "foo",
	}
	if i.FullString() != "foo" {
		t.Fatalf("err: %#v", i)
	}

	expected := map[string]InterpolatedVariable{
		"var.foo": v1,
		"var.bar": v2,
	}
	if !reflect.DeepEqual(i.Variables(), expected) {
		t.Fatalf("bad: %#v", i.Variables())
	}

	actual, err := i.Interpolate(map[string]string{
		"var.foo": "bar",
		"var.bar": "baz",
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if actual != "bar baz" {
		t.Fatalf("bad: %#v", actual)
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

func TestVariableInterpolation_impl(t *testing.T) {
	var _ Interpolation = new(VariableInterpolation)
}

func TestVariableInterpolation(t *testing.T) {
	uv, err := NewUserVariable("var.foo")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	i := &VariableInterpolation{Variable: uv, key: "var.foo"}
	if i.FullString() != "var.foo" {
		t.Fatalf("err: %#v", i)
	}

	expected := map[string]InterpolatedVariable{"var.foo": uv}
	if !reflect.DeepEqual(i.Variables(), expected) {
		t.Fatalf("bad: %#v", i.Variables())
	}

	actual, err := i.Interpolate(map[string]string{
		"var.foo": "bar",
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if actual != "bar" {
		t.Fatalf("bad: %#v", actual)
	}
}
