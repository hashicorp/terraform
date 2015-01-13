package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config/lang"
)

func TestInterpolateFuncFile(t *testing.T) {
	tf, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	path := tf.Name()
	tf.Write([]byte("foo"))
	tf.Close()
	defer os.Remove(path)

	testFunction(t, []testFunctionCase{
		{
			fmt.Sprintf(`${file("%s")}`, path),
			"foo",
			false,
		},

		// Invalid path
		{
			`${file("/i/dont/exist")}`,
			nil,
			true,
		},

		// Too many args
		{
			`${file("foo", "bar")}`,
			nil,
			true,
		},
	})
}

func TestInterpolateFuncJoin(t *testing.T) {
	testFunction(t, []testFunctionCase{
		{
			`${join(",")}`,
			nil,
			true,
		},

		{
			`${join(",", "foo")}`,
			"foo",
			false,
		},

		/*
			TODO
			{
				`${join(",", "foo", "bar")}`,
				"foo,bar",
				false,
			},
		*/

		{
			fmt.Sprintf(`${join(".", "%s")}`,
				fmt.Sprintf(
					"foo%sbar%sbaz",
					InterpSplitDelim,
					InterpSplitDelim)),
			"foo.bar.baz",
			false,
		},
	})
}

/*
func TestInterpolateFuncLookup(t *testing.T) {
	testFunction(t, []testFunctionCase{
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
*/

func TestInterpolateFuncElement(t *testing.T) {
	testFunction(t, []testFunctionCase{
		{
			fmt.Sprintf(`${element("%s", "1")}`,
				"foo"+InterpSplitDelim+"baz"),
			"baz",
			false,
		},

		{
			`${element("foo", "0")}`,
			"foo",
			false,
		},

		// Invalid index should wrap vs. out-of-bounds
		{
			fmt.Sprintf(`${element("%s", "2")}`,
				"foo"+InterpSplitDelim+"baz"),
			"foo",
			false,
		},

		// Too many args
		{
			fmt.Sprintf(`${element("%s", "0", "2")}`,
				"foo"+InterpSplitDelim+"baz"),
			nil,
			true,
		},
	})
}

type testFunctionCase struct {
	Input  string
	Result interface{}
	Error  bool
}

func testFunction(t *testing.T, cases []testFunctionCase) {
	for i, tc := range cases {
		ast, err := lang.Parse(tc.Input)
		if err != nil {
			t.Fatalf("%d: err: %s", i, err)
		}

		engine := langEngine(nil)
		out, _, err := engine.Execute(ast)
		if (err != nil) != tc.Error {
			t.Fatalf("%d: err: %s", i, err)
		}

		if !reflect.DeepEqual(out, tc.Result) {
			t.Fatalf("%d: bad: %#v", i, out)
		}
	}
}
