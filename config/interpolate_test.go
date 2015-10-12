package config

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config/lang"
)

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
		{
			"module.foo.bar",
			&ModuleVariable{
				Name:  "foo",
				Field: "bar",
				key:   "module.foo.bar",
			},
			false,
		},
		{
			"count.index",
			&CountVariable{
				Type: CountValueIndex,
				key:  "count.index",
			},
			false,
		},
		{
			"count.nope",
			&CountVariable{
				Type: CountValueInvalid,
				key:  "count.nope",
			},
			false,
		},
		{
			"path.module",
			&PathVariable{
				Type: PathValueModule,
				key:  "path.module",
			},
			false,
		},
		{
			"self.address",
			&SelfVariable{
				Field: "address",
				key:   "self.address",
			},
			false,
		},
	}

	for i, tc := range cases {
		actual, err := NewInterpolatedVariable(tc.Input)
		if err != nil != tc.Error {
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

func TestNewUserVariable_map(t *testing.T) {
	v, err := NewUserVariable("var.bar.baz")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if v.Name != "bar" {
		t.Fatalf("bad: %#v", v.Name)
	}
	if v.Elem != "baz" {
		t.Fatalf("bad: %#v", v.Elem)
	}
	if v.FullKey() != "var.bar.baz" {
		t.Fatalf("bad: %#v", v)
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

func TestDetectVariables(t *testing.T) {
	cases := []struct {
		Input  string
		Result []InterpolatedVariable
	}{
		{
			"foo $${var.foo}",
			nil,
		},

		{
			"foo ${var.foo}",
			[]InterpolatedVariable{
				&UserVariable{
					Name: "foo",
					key:  "var.foo",
				},
			},
		},

		{
			"foo ${var.foo} ${var.bar}",
			[]InterpolatedVariable{
				&UserVariable{
					Name: "foo",
					key:  "var.foo",
				},
				&UserVariable{
					Name: "bar",
					key:  "var.bar",
				},
			},
		},
	}

	for _, tc := range cases {
		ast, err := lang.Parse(tc.Input)
		if err != nil {
			t.Fatalf("%s\n\nInput: %s", err, tc.Input)
		}

		actual, err := DetectVariables(ast)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		if !reflect.DeepEqual(actual, tc.Result) {
			t.Fatalf("bad: %#v\n\nInput: %s", actual, tc.Input)
		}
	}
}
