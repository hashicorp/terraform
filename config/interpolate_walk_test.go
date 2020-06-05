package config

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/configs/hcl2shim"
	"github.com/mitchellh/reflectwalk"
)

func TestInterpolationWalker_detect(t *testing.T) {
	cases := []struct {
		Input  interface{}
		Result []string
	}{
		{
			Input: map[string]interface{}{
				"foo": "$${var.foo}",
			},
			Result: []string{
				"Literal(TypeString, ${var.foo})",
			},
		},

		{
			Input: map[string]interface{}{
				"foo": "${var.foo}",
			},
			Result: []string{
				"Variable(var.foo)",
			},
		},

		{
			Input: map[string]interface{}{
				"foo": "${aws_instance.foo.*.num}",
			},
			Result: []string{
				"Variable(aws_instance.foo.*.num)",
			},
		},

		{
			Input: map[string]interface{}{
				"foo": "${lookup(var.foo)}",
			},
			Result: []string{
				"Call(lookup, Variable(var.foo))",
			},
		},

		{
			Input: map[string]interface{}{
				"foo": `${file("test.txt")}`,
			},
			Result: []string{
				"Call(file, Literal(TypeString, test.txt))",
			},
		},

		{
			Input: map[string]interface{}{
				"foo": `${file("foo/bar.txt")}`,
			},
			Result: []string{
				"Call(file, Literal(TypeString, foo/bar.txt))",
			},
		},

		{
			Input: map[string]interface{}{
				"foo": `${join(",", foo.bar.*.id)}`,
			},
			Result: []string{
				"Call(join, Literal(TypeString, ,), Variable(foo.bar.*.id))",
			},
		},

		{
			Input: map[string]interface{}{
				"foo": `${concat("localhost", ":8080")}`,
			},
			Result: []string{
				"Call(concat, Literal(TypeString, localhost), Literal(TypeString, :8080))",
			},
		},
	}

	for i, tc := range cases {
		var actual []string
		detectFn := func(root ast.Node) (interface{}, error) {
			actual = append(actual, fmt.Sprintf("%s", root))
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
		Value  interface{}
	}{
		{
			Input: map[string]interface{}{
				"foo": "$${var.foo}",
			},
			Output: map[string]interface{}{
				"foo": "bar",
			},
			Value: "bar",
		},

		{
			Input: map[string]interface{}{
				"foo": "hello, ${var.foo}",
			},
			Output: map[string]interface{}{
				"foo": "bar",
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
			Value: []interface{}{"bar", "baz"},
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
					hcl2shim.UnknownVariableValue,
					"baz",
					"bing",
				},
			},
			Value: []interface{}{hcl2shim.UnknownVariableValue, "baz"},
		},
	}

	for i, tc := range cases {
		fn := func(ast.Node) (interface{}, error) {
			return tc.Value, nil
		}

		t.Run(fmt.Sprintf("walk-%d", i), func(t *testing.T) {
			w := &interpolationWalker{F: fn, Replace: true}
			if err := reflectwalk.Walk(tc.Input, w); err != nil {
				t.Fatalf("err: %s", err)
			}

			if !reflect.DeepEqual(tc.Input, tc.Output) {
				t.Fatalf("%d: bad:\n\nexpected:%#v\ngot:%#v", i, tc.Output, tc.Input)
			}
		})
	}
}
