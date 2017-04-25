package variables

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParseInput(t *testing.T) {
	cases := []struct {
		Name   string
		Input  string
		Result interface{}
		Error  bool
	}{
		{
			"unquoted string",
			"foo",
			"foo",
			false,
		},

		{
			"number",
			"1",
			"1",
			false,
		},

		{
			"float",
			"1.2",
			"1.2",
			false,
		},

		{
			"hex number",
			"0x12",
			"0x12",
			false,
		},

		{
			"bool",
			"true",
			"true",
			false,
		},

		{
			"list",
			`["foo"]`,
			[]interface{}{"foo"},
			false,
		},

		{
			"map",
			`{ foo = "bar" }`,
			map[string]interface{}{"foo": "bar"},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			actual, err := ParseInput(tc.Input)
			if (err != nil) != tc.Error {
				t.Fatalf("err: %s", err)
			}
			if err != nil {
				return
			}

			if !reflect.DeepEqual(actual, tc.Result) {
				t.Fatalf("bad: %#v", actual)
			}
		})
	}
}
