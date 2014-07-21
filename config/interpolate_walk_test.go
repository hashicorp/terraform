package config

import (
	"reflect"
	"testing"

	"github.com/mitchellh/reflectwalk"
)

func TestInterpolationWalker_detect(t *testing.T) {
	cases := []struct {
		Input  interface{}
		Result []Interpolation
	}{
		{
			Input: map[string]interface{}{
				"foo": "$${var.foo}",
			},
			Result: nil,
		},

		{
			Input: map[string]interface{}{
				"foo": "${var.foo}",
			},
			Result: []Interpolation{
				&VariableInterpolation{
					Variable: &UserVariable{
						Name: "foo",
						key:  "var.foo",
					},
					key: "var.foo",
				},
			},
		},
	}

	for i, tc := range cases {
		var actual []Interpolation

		detectFn := func(i Interpolation) (string, error) {
			actual = append(actual, i)
			return "", nil
		}

		w := &interpolationWalker{F: detectFn}
		if err := reflectwalk.Walk(tc.Input, w); err != nil {
			t.Fatalf("err: %s", err)
		}

		if !reflect.DeepEqual(actual, tc.Result) {
			t.Fatalf("%d: bad:\n\n%#v", i, actual)
		}
	}
}

func TestInterpolationWalker_replace(t *testing.T) {
	cases := []struct {
		Input  interface{}
		Output interface{}
	}{
		{
			Input: map[string]interface{}{
				"foo": "$${var.foo}",
			},
			Output: map[string]interface{}{
				"foo": "$${var.foo}",
			},
		},

		{
			Input: map[string]interface{}{
				"foo": "hello, ${var.foo}",
			},
			Output: map[string]interface{}{
				"foo": "hello, bar",
			},
		},
	}

	for i, tc := range cases {
		fn := func(i Interpolation) (string, error) {
			return "bar", nil
		}

		w := &interpolationWalker{F: fn, Replace: true}
		if err := reflectwalk.Walk(tc.Input, w); err != nil {
			t.Fatalf("err: %s", err)
		}

		if !reflect.DeepEqual(tc.Input, tc.Output) {
			t.Fatalf("%d: bad:\n\n%#v", i, tc.Input)
		}
	}
}
