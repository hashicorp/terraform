package config

import (
	"reflect"
	"testing"
)

func TestExprParse(t *testing.T) {
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
			},
			false,
		},

		{
			"lookup(var.foo, var.bar)",
			&FunctionInterpolation{
				Func: nil, // Funcs["lookup"]
				Args: []Interpolation{
					&VariableInterpolation{
						Variable: &UserVariable{
							Name: "foo",
							key:  "var.foo",
						},
					},
					&VariableInterpolation{
						Variable: &UserVariable{
							Name: "bar",
							key:  "var.bar",
						},
					},
				},
			},
			false,
		},
	}

	for i, tc := range cases {
		actual, err := ExprParse(tc.Input)
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
