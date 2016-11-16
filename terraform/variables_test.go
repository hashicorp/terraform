package terraform

import (
	"reflect"
	"testing"
)

func TestVariables(t *testing.T) {
	cases := map[string]struct {
		Module   string
		Env      map[string]string
		Override map[string]interface{}
		Error    bool
		Expected map[string]interface{}
	}{
		"config only": {
			"vars-basic",
			nil,
			nil,
			false,
			map[string]interface{}{
				"a": "foo",
				"b": []interface{}{},
				"c": map[string]interface{}{},
			},
		},

		"env vars": {
			"vars-basic",
			map[string]string{
				"TF_VAR_a": "bar",
				"TF_VAR_b": `["foo", "bar"]`,
				"TF_VAR_c": `{"foo" = "bar"}`,
			},
			nil,
			false,
			map[string]interface{}{
				"a": "bar",
				"b": []interface{}{"foo", "bar"},
				"c": map[string]interface{}{
					"foo": "bar",
				},
			},
		},

		"override": {
			"vars-basic",
			nil,
			map[string]interface{}{
				"a": "bar",
				"b": []interface{}{"foo", "bar"},
				"c": map[string]interface{}{
					"foo": "bar",
				},
			},
			false,
			map[string]interface{}{
				"a": "bar",
				"b": []interface{}{"foo", "bar"},
				"c": map[string]interface{}{
					"foo": "bar",
				},
			},
		},

		"override partial map": {
			"vars-basic",
			map[string]string{
				"TF_VAR_c": `{"foo" = "a", "bar" = "baz"}`,
			},
			map[string]interface{}{
				"c": map[string]interface{}{
					"foo": "bar",
				},
			},
			false,
			map[string]interface{}{
				"a": "foo",
				"b": []interface{}{},
				"c": map[string]interface{}{
					"foo": "bar",
					"bar": "baz",
				},
			},
		},

		"bools: config only": {
			"vars-basic-bool",
			nil,
			nil,
			false,
			map[string]interface{}{
				"a": "1",
				"b": "0",
			},
		},

		"bools: override with string": {
			"vars-basic-bool",
			nil,
			map[string]interface{}{
				"a": "foo",
				"b": "bar",
			},
			false,
			map[string]interface{}{
				"a": "foo",
				"b": "bar",
			},
		},

		"bools: override with env": {
			"vars-basic-bool",
			map[string]string{
				"TF_VAR_a": "false",
				"TF_VAR_b": "true",
			},
			nil,
			false,
			map[string]interface{}{
				"a": "false",
				"b": "true",
			},
		},

		"bools: override with bool": {
			"vars-basic-bool",
			nil,
			map[string]interface{}{
				"a": false,
				"b": true,
			},
			false,
			map[string]interface{}{
				"a": "0",
				"b": "1",
			},
		},

		"override map with string": {
			"vars-basic",
			map[string]string{
				"TF_VAR_c": `{"foo" = "a", "bar" = "baz"}`,
			},
			map[string]interface{}{
				"c": "bar",
			},
			true,
			nil,
		},
	}

	for name, tc := range cases {
		// Wrapped in a func so we can get defers to work
		t.Run(name, func(t *testing.T) {
			// Set the env vars
			for k, v := range tc.Env {
				defer tempEnv(t, k, v)()
			}

			m := testModule(t, tc.Module)
			actual, err := Variables(m, tc.Override)
			if (err != nil) != tc.Error {
				t.Fatalf("%s: err: %s", name, err)
			}
			if err != nil {
				return
			}

			if !reflect.DeepEqual(actual, tc.Expected) {
				t.Fatalf("%s\n\nexpected: %#v\n\ngot: %#v", name, tc.Expected, actual)
			}
		})
	}
}
