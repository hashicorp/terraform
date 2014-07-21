package config

import (
	"testing"
)

func TestInterpolateFuncLookup(t *testing.T) {
	cases := []struct {
		M      map[string]string
		Args   []string
		Result string
		Error  bool
	}{
		{
			map[string]string{
				"var.foo.bar": "baz",
			},
			[]string{"foo", "bar"},
			"baz",
			false,
		},

		// Invalid key
		{
			map[string]string{
				"var.foo.bar": "baz",
			},
			[]string{"foo", "baz"},
			"",
			true,
		},

		// Too many args
		{
			map[string]string{
				"var.foo.bar": "baz",
			},
			[]string{"foo", "bar", "baz"},
			"",
			true,
		},
	}

	for i, tc := range cases {
		actual, err := interpolationFuncLookup(tc.M, tc.Args...)
		if (err != nil) != tc.Error {
			t.Fatalf("%d: err: %s", i, err)
		}

		if actual != tc.Result {
			t.Fatalf("%d: bad: %#v", i, actual)
		}
	}
}
