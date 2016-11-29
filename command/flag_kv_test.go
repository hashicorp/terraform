package command

import (
	"flag"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestFlagStringKV_impl(t *testing.T) {
	var _ flag.Value = new(FlagStringKV)
}

func TestFlagStringKV(t *testing.T) {
	cases := []struct {
		Input  string
		Output map[string]string
		Error  bool
	}{
		{
			"key=value",
			map[string]string{"key": "value"},
			false,
		},

		{
			"key=",
			map[string]string{"key": ""},
			false,
		},

		{
			"key=foo=bar",
			map[string]string{"key": "foo=bar"},
			false,
		},

		{
			"map.key=foo",
			map[string]string{"map.key": "foo"},
			false,
		},

		{
			"key",
			nil,
			true,
		},

		{
			"key=/path",
			map[string]string{"key": "/path"},
			false,
		},
	}

	for _, tc := range cases {
		f := new(FlagStringKV)
		err := f.Set(tc.Input)
		if err != nil != tc.Error {
			t.Fatalf("bad error. Input: %#v\n\nError: %s", tc.Input, err)
		}

		actual := map[string]string(*f)
		if !reflect.DeepEqual(actual, tc.Output) {
			t.Fatalf("bad: %#v", actual)
		}
	}
}

func TestFlagTypedKV_impl(t *testing.T) {
	var _ flag.Value = new(FlagTypedKV)
}

func TestFlagTypedKV(t *testing.T) {
	cases := []struct {
		Input  string
		Output map[string]interface{}
		Error  bool
	}{
		{
			"key=value",
			map[string]interface{}{"key": "value"},
			false,
		},

		{
			"key=",
			map[string]interface{}{"key": ""},
			false,
		},

		{
			"key=foo=bar",
			map[string]interface{}{"key": "foo=bar"},
			false,
		},

		{
			"key=false",
			map[string]interface{}{"key": "false"},
			false,
		},

		{
			"map.key=foo",
			map[string]interface{}{"map.key": "foo"},
			false,
		},

		{
			"key",
			nil,
			true,
		},

		{
			`key=["hello", "world"]`,
			map[string]interface{}{"key": []interface{}{"hello", "world"}},
			false,
		},

		{
			`key={"hello" = "world", "foo" = "bar"}`,
			map[string]interface{}{
				"key": map[string]interface{}{
					"hello": "world",
					"foo":   "bar",
				},
			},
			false,
		},

		{
			`key={"hello" = "world", "foo" = "bar"}\nkey2="invalid"`,
			nil,
			true,
		},

		{
			"key=/path",
			map[string]interface{}{"key": "/path"},
			false,
		},

		{
			"key=1234.dkr.ecr.us-east-1.amazonaws.com/proj:abcdef",
			map[string]interface{}{"key": "1234.dkr.ecr.us-east-1.amazonaws.com/proj:abcdef"},
			false,
		},

		// simple values that can parse as numbers should remain strings
		{
			"key=1",
			map[string]interface{}{
				"key": "1",
			},
			false,
		},
		{
			"key=1.0",
			map[string]interface{}{
				"key": "1.0",
			},
			false,
		},
		{
			"key=0x10",
			map[string]interface{}{
				"key": "0x10",
			},
			false,
		},
	}

	for _, tc := range cases {
		f := new(FlagTypedKV)
		err := f.Set(tc.Input)
		if err != nil != tc.Error {
			t.Fatalf("bad error. Input: %#v\n\nError: %s", tc.Input, err)
		}

		actual := map[string]interface{}(*f)
		if !reflect.DeepEqual(actual, tc.Output) {
			t.Fatalf("bad:\nexpected: %s\n\n     got: %s\n", spew.Sdump(tc.Output), spew.Sdump(actual))
		}
	}
}

func TestFlagKVFile_impl(t *testing.T) {
	var _ flag.Value = new(FlagKVFile)
}

func TestFlagKVFile(t *testing.T) {
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
		Input  string
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
	}

	path := testTempFile(t)

	for _, tc := range cases {
		if err := ioutil.WriteFile(path, []byte(tc.Input), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}

		f := new(FlagKVFile)
		err := f.Set(path)
		if err != nil != tc.Error {
			t.Fatalf("bad error. Input: %#v, err: %s", tc.Input, err)
		}

		actual := map[string]interface{}(*f)
		if !reflect.DeepEqual(actual, tc.Output) {
			t.Fatalf("bad: %#v", actual)
		}
	}
}
