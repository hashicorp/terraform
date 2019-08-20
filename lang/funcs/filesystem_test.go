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
		Err  bool
	}{
		{
			cty.StringVal("testdata/hello.txt"),
			cty.EmptyObjectVal,
			cty.StringVal("Hello World"),
			false,
		},
		{
			cty.StringVal("testdata/icon.png"),
			cty.EmptyObjectVal,
			cty.NilVal,
			true, // Not valid UTF-8
		},
		{
			cty.StringVal("testdata/missing"),
			cty.EmptyObjectVal,
			cty.NilVal,
			true, // no file exists
		},
		{
			cty.StringVal("testdata/hello.tmpl"),
			cty.MapVal(map[string]cty.Value{
				"name": cty.StringVal("Jodie"),
			}),
			cty.StringVal("Hello, Jodie!"),
			false,
		},
		{
			cty.StringVal("testdata/hello.tmpl"),
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("Jimbo"),
			}),
			cty.StringVal("Hello, Jimbo!"),
			false,
		},
		{
			cty.StringVal("testdata/hello.tmpl"),
			cty.EmptyObjectVal,
			cty.NilVal,
			true, // "name" is missing from the vars map
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
			false,
		},
		{
			cty.StringVal("testdata/recursive.tmpl"),
			cty.MapValEmpty(cty.String),
			cty.NilVal,
			true, // recursive templatefile call not allowed
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
			false,
		},
		{
			cty.StringVal("testdata/list.tmpl"),
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.True,
			}),
			cty.NilVal,
			true, // iteration over non-iterable value
		},
		{
			cty.StringVal("testdata/bare.tmpl"),
			cty.ObjectVal(map[string]cty.Value{
				"val": cty.True,
			}),
			cty.True, // since this template contains only an interpolation, its true value shines through
			false,
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
		Pattern cty.Value
		Want    cty.Value
		Err     bool
	}{
		{
			cty.StringVal("testdata/missing"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("testdata/missing*"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("*/missing"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("testdata"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("testdata*"),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			cty.StringVal("testdata/*.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("testdata/hello.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("testdata/hello.???"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("testdata/hello*"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.tmpl"),
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("*/hello.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("*/*.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("*/hello*"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.tmpl"),
				cty.StringVal("testdata/hello.txt"),
			}),
			false,
		},
		{
			cty.StringVal("["),
			cty.SetValEmpty(cty.String),
			true,
		},
		{
			cty.StringVal("\\"),
			cty.SetValEmpty(cty.String),
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("FileSet(\".\", %#v)", test.Pattern), func(t *testing.T) {
			got, err := FileSet(".", test.Pattern)

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
