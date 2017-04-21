package variables

import (
	"flag"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
)

func TestFlagFile_impl(t *testing.T) {
	var _ flag.Value = new(FlagFile)
}

func TestFlagFile(t *testing.T) {
	inputLibucl := `
foo = "bar"
`
	inputMap := `
foo = {
	k = "v"
}`

	inputJson := `{
		"foo": "bar"}`

	cases := []struct {
		Input  interface{}
		Output map[string]interface{}
		Error  bool
	}{
		{
			inputLibucl,
			map[string]interface{}{"foo": "bar"},
			false,
		},

		{
			inputJson,
			map[string]interface{}{"foo": "bar"},
			false,
		},

		{
			`map.key = "foo"`,
			map[string]interface{}{"map.key": "foo"},
			false,
		},

		{
			inputMap,
			map[string]interface{}{
				"foo": map[string]interface{}{
					"k": "v",
				},
			},
			false,
		},

		{
			[]string{
				`foo = { "k" = "v"}`,
				`foo = { "j" = "v" }`,
			},
			map[string]interface{}{
				"foo": map[string]interface{}{
					"k": "v",
					"j": "v",
				},
			},
			false,
		},
	}

	path := testTempFile(t)

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			var input []string
			switch i := tc.Input.(type) {
			case string:
				input = []string{i}
			case []string:
				input = i
			default:
				t.Fatalf("bad input type: %T", i)
			}

			f := new(FlagFile)
			for _, input := range input {
				if err := ioutil.WriteFile(path, []byte(input), 0644); err != nil {
					t.Fatalf("err: %s", err)
				}

				err := f.Set(path)
				if err != nil != tc.Error {
					t.Fatalf("bad error. Input: %#v, err: %s", input, err)
				}
			}

			actual := map[string]interface{}(*f)
			if !reflect.DeepEqual(actual, tc.Output) {
				t.Fatalf("bad: %#v", actual)
			}
		})
	}
}
