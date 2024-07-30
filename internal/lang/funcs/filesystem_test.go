// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/lang/marks"
)

func TestFile(t *testing.T) {
	tests := []struct {
		Path cty.Value
		Want cty.Value
		Err  string
	}{
		{
			cty.StringVal("testdata/hello.txt"),
			cty.StringVal("Hello World"),
			``,
		},
		{
			cty.StringVal("testdata/icon.png"),
			cty.NilVal,
			`contents of "testdata/icon.png" are not valid UTF-8; use the filebase64 function to obtain the Base64 encoded contents or the other file functions (e.g. filemd5, filesha256) to obtain file hashing results instead`,
		},
		{
			cty.StringVal("testdata/icon.png").Mark(marks.Sensitive),
			cty.NilVal,
			`contents of (sensitive value) are not valid UTF-8; use the filebase64 function to obtain the Base64 encoded contents or the other file functions (e.g. filemd5, filesha256) to obtain file hashing results instead`,
		},
		{
			cty.StringVal("testdata/missing"),
			cty.NilVal,
			`no file exists at "testdata/missing"; this function works only with files that are distributed as part of the configuration source code, so if this file will be created by a resource in this configuration you must instead obtain this result from an attribute of that resource`,
		},
		{
			cty.StringVal("testdata/missing").Mark(marks.Sensitive),
			cty.NilVal,
			`no file exists at (sensitive value); this function works only with files that are distributed as part of the configuration source code, so if this file will be created by a resource in this configuration you must instead obtain this result from an attribute of that resource`,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("File(\".\", %#v)", test.Path), func(t *testing.T) {
			got, err := File(".", test.Path)

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
			`contents of "testdata/icon.png" are not valid UTF-8; use the filebase64 function to obtain the Base64 encoded contents or the other file functions (e.g. filemd5, filesha256) to obtain file hashing results instead`,
		},
		{
			cty.StringVal("testdata/missing"),
			cty.EmptyObjectVal,
			cty.NilVal,
			`no file exists at "testdata/missing"; this function works only with files that are distributed as part of the configuration source code, so if this file will be created by a resource in this configuration you must instead obtain this result from an attribute of that resource`,
		},
		{
			cty.StringVal("testdata/secrets.txt").Mark(marks.Sensitive),
			cty.EmptyObjectVal,
			cty.NilVal,
			`no file exists at (sensitive value); this function works only with files that are distributed as part of the configuration source code, so if this file will be created by a resource in this configuration you must instead obtain this result from an attribute of that resource`,
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
			`testdata/recursive.tmpl:1,3-16: Error in function call; Call to function "templatefile" failed: cannot recursively call templatefile from inside another template function.`,
		},
		{
			cty.StringVal("testdata/recursive_namespaced.tmpl"),
			cty.MapValEmpty(cty.String),
			cty.NilVal,
			`testdata/recursive_namespaced.tmpl:1,3-22: Error in function call; Call to function "core::templatefile" failed: cannot recursively call templatefile from inside another template function.`,
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
		{
			// If the template filename is sensitive then we also treat the
			// rendered result as sensitive, because the rendered result
			// is likely to imply which filename was used.
			// (Sensitive filenames seem pretty unlikely, but if they do
			// crop up then we should handle them consistently with our
			// usual sensitivity rules.)
			cty.StringVal("testdata/hello.txt").Mark(marks.Sensitive),
			cty.EmptyObjectVal,
			cty.StringVal("Hello World").Mark(marks.Sensitive),
			``,
		},
	}

	funcs := map[string]function.Function{
		"join":       stdlib.JoinFunc,
		"core::join": stdlib.JoinFunc,
	}
	funcsFunc := func() (funcTable map[string]function.Function, fsFuncs collections.Set[string], templateFuncs collections.Set[string]) {
		return funcs, collections.NewSetCmp[string](), collections.NewSetCmp[string]("templatefile")
	}
	templateFileFn := MakeTemplateFileFunc(".", funcsFunc)
	funcs["templatefile"] = templateFileFn
	funcs["core::templatefile"] = templateFileFn

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
		Err  string
	}{
		{
			cty.StringVal("testdata/hello.txt"),
			cty.BoolVal(true),
			``,
		},
		{
			cty.StringVal(""),
			cty.BoolVal(false),
			`"." is a directory, not a file`,
		},
		{
			cty.StringVal("testdata").Mark(marks.Sensitive),
			cty.BoolVal(false),
			`(sensitive value) is a directory, not a file`,
		},
		{
			cty.StringVal("testdata/missing"),
			cty.BoolVal(false),
			``,
		},
		{
			cty.StringVal("testdata/unreadable/foobar"),
			cty.BoolVal(false),
			`failed to stat "testdata/unreadable/foobar"`,
		},
		{
			cty.StringVal("testdata/unreadable/foobar").Mark(marks.Sensitive),
			cty.BoolVal(false),
			`failed to stat (sensitive value)`,
		},
	}

	// Ensure "unreadable" directory cannot be listed during the test run
	fi, err := os.Lstat("testdata/unreadable")
	if err != nil {
		t.Fatal(err)
	}
	os.Chmod("testdata/unreadable", 0000)
	defer func(mode os.FileMode) {
		os.Chmod("testdata/unreadable", mode)
	}(fi.Mode())

	for _, test := range tests {
		t.Run(fmt.Sprintf("FileExists(\".\", %#v)", test.Path), func(t *testing.T) {
			got, err := FileExists(".", test.Path)

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

func TestFileSet(t *testing.T) {
	tests := []struct {
		Path    cty.Value
		Pattern cty.Value
		Want    cty.Value
		Err     string
	}{
		{
			cty.StringVal("."),
			cty.StringVal("testdata*"),
			cty.SetValEmpty(cty.String),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata"),
			cty.SetValEmpty(cty.String),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("{testdata,missing}"),
			cty.SetValEmpty(cty.String),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/missing"),
			cty.SetValEmpty(cty.String),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/missing*"),
			cty.SetValEmpty(cty.String),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("*/missing"),
			cty.SetValEmpty(cty.String),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("**/missing"),
			cty.SetValEmpty(cty.String),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/*.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/hello.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/hello.???"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/hello*"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.tmpl"),
				cty.StringVal("testdata/hello.txt"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/hello.{tmpl,txt}"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.tmpl"),
				cty.StringVal("testdata/hello.txt"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("*/hello.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("*/*.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.txt"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("*/hello*"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.tmpl"),
				cty.StringVal("testdata/hello.txt"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("**/hello*"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.tmpl"),
				cty.StringVal("testdata/hello.txt"),
				cty.StringVal("testdata/somedir/hello.txt"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("**/hello.{tmpl,txt}"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/hello.tmpl"),
				cty.StringVal("testdata/hello.txt"),
				cty.StringVal("testdata/somedir/hello.txt"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("["),
			cty.SetValEmpty(cty.String),
			`failed to glob pattern "[": syntax error in pattern`,
		},
		{
			cty.StringVal("."),
			cty.StringVal("[").Mark(marks.Sensitive),
			cty.SetValEmpty(cty.String),
			`failed to glob pattern (sensitive value): syntax error in pattern`,
		},
		{
			cty.StringVal("."),
			cty.StringVal("\\"),
			cty.SetValEmpty(cty.String),
			`failed to glob pattern "\\": syntax error in pattern`,
		},
		{
			cty.StringVal("testdata"),
			cty.StringVal("missing"),
			cty.SetValEmpty(cty.String),
			``,
		},
		{
			cty.StringVal("testdata"),
			cty.StringVal("missing*"),
			cty.SetValEmpty(cty.String),
			``,
		},
		{
			cty.StringVal("testdata"),
			cty.StringVal("*.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("hello.txt"),
			}),
			``,
		},
		{
			cty.StringVal("testdata"),
			cty.StringVal("hello.txt"),
			cty.SetVal([]cty.Value{
				cty.StringVal("hello.txt"),
			}),
			``,
		},
		{
			cty.StringVal("testdata"),
			cty.StringVal("hello.???"),
			cty.SetVal([]cty.Value{
				cty.StringVal("hello.txt"),
			}),
			``,
		},
		{
			cty.StringVal("testdata"),
			cty.StringVal("hello*"),
			cty.SetVal([]cty.Value{
				cty.StringVal("hello.tmpl"),
				cty.StringVal("hello.txt"),
			}),
			``,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("FileSet(\".\", %#v, %#v)", test.Path, test.Pattern), func(t *testing.T) {
			got, err := FileSet(".", test.Path, test.Pattern)

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

func TestDirectoryExists(t *testing.T) {
	tests := []struct {
		Path cty.Value
		Want cty.Value
		Err  string
	}{
		{
			cty.StringVal("testdata/"),
			cty.BoolVal(true),
			``,
		},
		{
			cty.StringVal(""),
			cty.BoolVal(true),
			``,
		},
		{
			cty.StringVal("missing"),
			cty.BoolVal(false),
			``,
		},
		{
			cty.StringVal("hello.txt"),
			cty.BoolVal(false),
			``,
		},
		{
			cty.StringVal("testdata/hello.txt"),
			cty.BoolVal(false),
			`"testdata/hello.txt" is not a directory`,
		},
		{
			cty.StringVal("testdata/unreadable/foobar"),
			cty.BoolVal(false),
			`failed to stat "testdata/unreadable/foobar"`,
		},
		{
			cty.StringVal("testdata/unreadable/foobar").Mark(marks.Sensitive),
			cty.BoolVal(false),
			`failed to stat (sensitive value)`,
		},
	}

	// Ensure "unreadable" directory cannot be listed during the test run
	fi, err := os.Lstat("testdata/unreadable")
	if err != nil {
		t.Fatal(err)
	}
	os.Chmod("testdata/unreadable", 0000)
	defer func(mode os.FileMode) {
		os.Chmod("testdata/unreadable", mode)
	}(fi.Mode())

	for _, test := range tests {
		t.Run(fmt.Sprintf("DirectoryExists(\".\", %#v)", test.Path), func(t *testing.T) {
			got, err := DirectoryExists(".", test.Path)

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

func TestDirectorySet(t *testing.T) {
	tests := []struct {
		Path    cty.Value
		Pattern cty.Value
		Want    cty.Value
		Err     string
	}{
		{
			cty.StringVal("."),
			cty.StringVal("*"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("**"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata"),
				cty.StringVal("testdata/somedir"),
				cty.StringVal("testdata/somedir/someotherdir"),
				cty.StringVal("testdata/unreadable"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("testdata/somedir/bogus"),
			cty.SetValEmpty(cty.String),
			``,
		},
		{
			cty.StringVal("./testdata"),
			cty.StringVal("**"),
			cty.SetVal([]cty.Value{
				cty.StringVal("somedir"),
				cty.StringVal("somedir/someotherdir"),
				cty.StringVal("unreadable"),
			}),
			``,
		},
		{
			cty.StringVal("./testdata"),
			cty.StringVal("**"),
			cty.SetVal([]cty.Value{
				cty.StringVal("somedir"),
				cty.StringVal("somedir/someotherdir"),
				cty.StringVal("unreadable"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("**/someother*"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/somedir/someotherdir"),
			}),
			``,
		},
		{
			cty.StringVal("."),
			cty.StringVal("**/{somedir,someotherdir}"),
			cty.SetVal([]cty.Value{
				cty.StringVal("testdata/somedir"),
				cty.StringVal("testdata/somedir/someotherdir"),
			}),
			``,
		},
		{
			cty.StringVal("./testdata/unreadable"),
			cty.StringVal("*"),
			cty.SetValEmpty(cty.String),
			``,
		},
	}

	// Ensure "unreadable" directory cannot be listed during the test run
	fi, err := os.Lstat("testdata/unreadable")
	if err != nil {
		t.Fatal(err)
	}
	os.Chmod("testdata/unreadable", 0000)
	defer func(mode os.FileMode) {
		os.Chmod("testdata/unreadable", mode)
	}(fi.Mode())

	for _, test := range tests {
		t.Run(fmt.Sprintf("DirectorySet(\".\", %#v, %#v)", test.Path, test.Pattern), func(t *testing.T) {
			got, err := DirectorySet(".", test.Path, test.Pattern)

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
