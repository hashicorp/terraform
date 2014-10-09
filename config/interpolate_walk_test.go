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
				},
			},
		},

		{
			Input: map[string]interface{}{
				"foo": "${aws_instance.foo.*.num}",
			},
			Result: []Interpolation{
				&VariableInterpolation{
					Variable: &ResourceVariable{
						Type:  "aws_instance",
						Name:  "foo",
						Field: "num",

						Multi: true,
						Index: -1,

						key: "aws_instance.foo.*.num",
					},
				},
			},
		},

		{
			Input: map[string]interface{}{
				"foo": "${lookup(var.foo)}",
			},
			Result: []Interpolation{
				&FunctionInterpolation{
					Func: nil,
					Args: []Interpolation{
						&VariableInterpolation{
							Variable: &UserVariable{
								Name: "foo",
								key:  "var.foo",
							},
						},
					},
				},
			},
		},

		{
			Input: map[string]interface{}{
				"foo": `${file("test.txt")}`,
			},
			Result: []Interpolation{
				&FunctionInterpolation{
					Func: nil,
					Args: []Interpolation{
						&LiteralInterpolation{
							Literal: "test.txt",
						},
					},
				},
			},
		},

		{
			Input: map[string]interface{}{
				"foo": `${file("foo/bar.txt")}`,
			},
			Result: []Interpolation{
				&FunctionInterpolation{
					Func: nil,
					Args: []Interpolation{
						&LiteralInterpolation{
							Literal: "foo/bar.txt",
						},
					},
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

		for _, a := range actual {
			// This is jank, but reflect.DeepEqual never has functions
			// being the same.
			if f, ok := a.(*FunctionInterpolation); ok {
				f.Func = nil
			}
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
		Value  string
	}{
		{
			Input: map[string]interface{}{
				"foo": "$${var.foo}",
			},
			Output: map[string]interface{}{
				"foo": "$${var.foo}",
			},
			Value: "bar",
		},

		{
			Input: map[string]interface{}{
				"foo": "hello, ${var.foo}",
			},
			Output: map[string]interface{}{
				"foo": "hello, bar",
			},
			Value: "bar",
		},

		{
			Input: map[string]interface{}{
				"foo": map[string]interface{}{
					"${var.foo}": "bar",
				},
			},
			Output: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "bar",
				},
			},
			Value: "bar",
		},

		{
			Input: map[string]interface{}{
				"foo": []interface{}{
					"${var.foo}",
					"bing",
				},
			},
			Output: map[string]interface{}{
				"foo": []interface{}{
					"bar",
					"baz",
					"bing",
				},
			},
			Value: "bar" + InterpSplitDelim + "baz",
		},
	}

	for i, tc := range cases {
		fn := func(i Interpolation) (string, error) {
			return tc.Value, nil
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
