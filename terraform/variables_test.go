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
	}

	for name, tc := range cases {
		if name != "override partial map" {
			continue
		}

		// Wrapped in a func so we can get defers to work
		func() {
			// Set the env vars
			for k, v := range tc.Env {
				defer tempEnv(t, k, v)()
			}

			m := testModule(t, tc.Module)
			actual, err := Variables(m, tc.Override)
			if (err != nil) != tc.Error {
				t.Fatalf("%s: err: %s", err)
			}
			if err != nil {
				return
			}

			if !reflect.DeepEqual(actual, tc.Expected) {
				t.Fatalf("%s: expected: %#v\n\ngot: %#v", name, tc.Expected, actual)
			}
		}()
	}
}
