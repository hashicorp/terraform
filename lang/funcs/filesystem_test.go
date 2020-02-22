package funcs

import (
	"fmt"
	"path/filepath"
	"testing"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

func TestFile(t *testing.T) {
	tests := []struct {
		Path cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.StringVal("testdata/hello.txt"),
			cty.StringVal("Hello World"),
			false,
		},
		{
			cty.StringVal("testdata/icon.png"),
			cty.NilVal,
			true, // Not valid UTF-8
		},
		{
			cty.StringVal("testdata/missing"),
			cty.NilVal,
			true, // no file exists
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("File(\".\", %#v)", test.Path), func(t *testing.T) {
			got, err := File(".", test.Path)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestTemplateFile(t *testing.T) {
	tests := []struct {
		Path cty.Value
		Vars cty.Value
		Want cty.Value
		Err  string
	}{
		{
			cty.StringVal("testdata/hello.txt"),
			cty.EmptyObjectVal,
			cty.StringVal("Hello World"),
			``,
		},
		{
			cty.StringVal("testdata/icon.png"),
			cty.EmptyObjectVal,
			cty.NilVal,
			`contents of testdata/icon.png are not valid UTF-8; use the filebase64 function to obtain the Base64 encoded contents or the other file functions (e.g. filemd5, filesha256) to obtain file hashing results instead`,
		},
		{
			cty.StringVal("testdata/missing"),
			cty.EmptyObjectVal,
			cty.NilVal,
			`no file exists at testdata/missing`,
		},
		{
			cty.StringVal("testdata/hello.tmpl"),
			cty.MapVal(map[string]cty.Value{
				"name": cty.StringVal("Jodie"),
			}),
			cty.StringVal("Hello, Jodie!"),
			``,
		},
		{
			cty.StringVal("testdata/hello.tmpl"),
			cty.MapVal(map[string]cty.Value{
				"name!": cty.StringVal("Jodie"),
			}),
			cty.NilVal,
			`invalid template variable name "name!": must start with a letter, followed by zero or more letters, digits, and underscores`,
		},
		{
			cty.StringVal("testdata/hello.tmpl"),
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("Jimbo"),
			}),
			cty.StringVal("Hello, Jimbo!"),
			``,
		},
		{
			cty.StringVal("testdata/hello.tmpl"),
			cty.EmptyObjectVal,
			cty.NilVal,
			`vars map does not contain key "name", referenced at testdata/hello.tmpl:1,10-14`,
		},
		{
			cty.StringVal("testdata/func.tmpl"),
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
					cty.StringVal("c"),
				}),
			}),
			cty.StringVal("The items are a, b, c"),
			``,
		},
		{
			cty.StringVal("testdata/recursive.tmpl"),
			cty.MapValEmpty(cty.String),
			cty.NilVal,
			`testdata/recursive.tmpl:1,3-16: Error in function call; Call to function "templatefile" failed: cannot recursively call templatefile from inside templatefile call.`,
		},
		{
			cty.StringVal("testdata/list.tmpl"),
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{
					cty.StringVal("a"),
					cty.StringVal("b"),
					cty.StringVal("c"),
				}),
			}),
			cty.StringVal("- a\n- b\n- c\n"),
			``,
		},
		{
			cty.StringVal("testdata/list.tmpl"),
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.True,
			}),
			cty.NilVal,
			`testdata/list.tmpl:1,13-17: Iteration over non-iterable value; A value of type bool cannot be used as the collection in a 'for' expression.`,
		},
		{
			cty.StringVal("testdata/bare.tmpl"),
			cty.ObjectVal(map[string]cty.Value{
				"val": cty.True,
			}),
			cty.True, // since this template contains only an interpolation, its true value shines through
			``,
		},
	}

	templateFileFn := MakeTemplateFileFunc(".", func() map[string]function.Function {
		return map[string]function.Function{
			"join":         JoinFunc,
			"templatefile": MakeFileFunc(".", false), // just a placeholder, since templatefile itself overrides this
		}
	})

	for _, test := range tests {
		t.Run(fmt.Sprintf("TemplateFile(%#v, %#v)", test.Path, test.Vars), func(t *testing.T) {
			got, err := templateFileFn.Call([]cty.Value{test.Path, test.Vars})

			if argErr, ok := err.(function.ArgError); ok {
				if argErr.Index < 0 || argErr.Index > 1 {
					t.Errorf("ArgError index %d is out of range for templatefile (must be 0 or 1)", argErr.Index)
				}
			}

			if test.Err != "" {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				if got, want := err.Error(), test.Err; got != want {
					t.Errorf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tests := []struct {
		Path cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.StringVal("testdata/hello.txt"),
			cty.BoolVal(true),
			false,
		},
		{
			cty.StringVal(""), // empty path
			cty.BoolVal(false),
			true,
		},
		{
			cty.StringVal("testdata/missing"),
			cty.BoolVal(false),
			false, // no file exists
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("FileExists(\".\", %#v)", test.Path), func(t *testing.T) {
			got, err := FileExists(".", test.Path)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestFileSet(t *testing.T) {
	tests := []struct {
		Path    cty.Value
		Pattern cty.Value
		Want    cty.Value
		Err     bool
	}{
		{
			cty.StringVal("."),
			cty.StringVal("testdata*"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("{testdata,missing}"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/missing"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/missing*"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("*/missing"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("**/missing"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/*.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/hello.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/hello.???"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/hello*"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.tmpl"),
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/hello.{tmpl,txt}"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.tmpl"),
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("*/hello.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("*/*.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("*/hello*"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.tmpl"),
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("**/hello*"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.tmpl"),
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("**/hello.{tmpl,txt}"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.tmpl"),
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("."),
			cty.StringVal("["),
			cty.SetValEmpty(cty.String),
			true,
		},
		{
			cty.StringVal("."),
			cty.StringVal("\\"),
			cty.SetValEmpty(cty.String),
			true,
		},
		{
			cty.StringVal("testdata"),
			cty.StringVal("missing"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("testdata"),
			cty.StringVal("missing*"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("testdata"),
			cty.StringVal("*.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("testdata"),
			cty.StringVal("hello.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("testdata"),
			cty.StringVal("hello.???"),
			cty.SetVal([]cty.Value{
				cty.StringVal("hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("testdata"),
			cty.StringVal("hello*"),
			cty.SetVal([]cty.Value{
				cty.StringVal("hello.tmpl"),
				cty.StringVal("hello.txt"),
			}),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("FileSet(\".\", %#v, %#v)", test.Path, test.Pattern), func(t *testing.T) {
			got, err := FileSet(".", test.Path, test.Pattern)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestFileBase64(t *testing.T) {
	tests := []struct {
		Path cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.StringVal("testdata/hello.txt"),
			cty.StringVal("SGVsbG8gV29ybGQ="),
			false,
		},
		{
			cty.StringVal("testdata/icon.png"),
			cty.StringVal("iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAMAAAAoLQ9TAAAABGdBTUEAALGPC/xhBQAAACBjSFJNAAB6JgAAgIQAAPoAAACA6AAAdTAAAOpgAAA6mAAAF3CculE8AAAAq1BMVEX///9cTuVeUeRcTuZcTuZcT+VbSe1cTuVdT+MAAP9JSbZcT+VcTuZAQLFAQLJcTuVcTuZcUuBBQbA/P7JAQLJaTuRcT+RcTuVGQ7xAQLJVVf9cTuVcTuVGRMFeUeRbTeJcTuU/P7JeTeZbTOVcTeZAQLJBQbNAQLNaUORcTeZbT+VcTuRAQLNAQLRdTuRHR8xgUOdgUN9cTuVdTeRdT+VZTulcTuVAQLL///8+GmETAAAANnRSTlMApibw+osO6DcBB3fIX87+oRk3yehB0/Nj/gNs7nsTRv3dHmu//JYUMLVr3bssjxkgEK5CaxeK03nIAAAAAWJLR0QAiAUdSAAAAAlwSFlzAAADoQAAA6EBvJf9gwAAAAd0SU1FB+EEBRIQDxZNTKsAAACCSURBVBjTfc7JFsFQEATQQpCYxyBEzJ55rvf/f0ZHcyQLvelTd1GngEwWycs5+UISyKLraSi9geWKK9Gr1j7AeqOJVtt2XtD1Bchef2BjQDAcCTC0CsA4mihMtXw2XwgsV2sFw812F+4P3y2GdI6nn3FGSs//4HJNAXDzU4Dg/oj/E+bsEbhf5cMsAAAAJXRFWHRkYXRlOmNyZWF0ZQAyMDE3LTA0LTA1VDE4OjE2OjE1KzAyOjAws5bLVQAAACV0RVh0ZGF0ZTptb2RpZnkAMjAxNy0wNC0wNVQxODoxNjoxNSswMjowMMLLc+kAAAAZdEVYdFNvZnR3YXJlAHd3dy5pbmtzY2FwZS5vcmeb7jwaAAAAC3RFWHRUaXRsZQBHcm91cJYfIowAAABXelRYdFJhdyBwcm9maWxlIHR5cGUgaXB0YwAAeJzj8gwIcVYoKMpPy8xJ5VIAAyMLLmMLEyMTS5MUAxMgRIA0w2QDI7NUIMvY1MjEzMQcxAfLgEigSi4A6hcRdPJCNZUAAAAASUVORK5CYII="),
			false,
		},
		{
			cty.StringVal("testdata/missing"),
			cty.NilVal,
			true, // no file exists
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("FileBase64(\".\", %#v)", test.Path), func(t *testing.T) {
			got, err := FileBase64(".", test.Path)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestBasename(t *testing.T) {
	tests := []struct {
		Path cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.StringVal("testdata/hello.txt"),
			cty.StringVal("hello.txt"),
			false,
		},
		{
			cty.StringVal("hello.txt"),
			cty.StringVal("hello.txt"),
			false,
		},
		{
			cty.StringVal(""),
			cty.StringVal("."),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Basename(%#v)", test.Path), func(t *testing.T) {
			got, err := Basename(test.Path)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestDirname(t *testing.T) {
	tests := []struct {
		Path cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.StringVal("testdata/hello.txt"),
			cty.StringVal("testdata"),
			false,
		},
		{
			cty.StringVal("testdata/foo/hello.txt"),
			cty.StringVal("testdata/foo"),
			false,
		},
		{
			cty.StringVal("hello.txt"),
			cty.StringVal("."),
			false,
		},
		{
			cty.StringVal(""),
			cty.StringVal("."),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Dirname(%#v)", test.Path), func(t *testing.T) {
			got, err := Dirname(test.Path)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestPathExpand(t *testing.T) {
	homePath, err := homedir.Dir()
	if err != nil {
		t.Fatalf("Error getting home directory: %v", err)
	}

	tests := []struct {
		Path cty.Value
		Want cty.Value
		Err  bool
	}{
		{
			cty.StringVal("~/test-file"),
			cty.StringVal(filepath.Join(homePath, "test-file")),
			false,
		},
		{
			cty.StringVal("~/another/test/file"),
			cty.StringVal(filepath.Join(homePath, "another/test/file")),
			false,
		},
		{
			cty.StringVal("/root/file"),
			cty.StringVal("/root/file"),
			false,
		},
		{
			cty.StringVal("/"),
			cty.StringVal("/"),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Dirname(%#v)", test.Path), func(t *testing.T) {
			got, err := Pathexpand(test.Path)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
