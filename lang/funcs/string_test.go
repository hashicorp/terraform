package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestJoin(t *testing.T) {
	tests := []struct {
		Sep   cty.Value
		Lists []cty.Value
		Want  cty.Value
	}{
		{
			cty.StringVal(" "),
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("Hello"),
					cty.StringVal("World"),
				}),
			},
			cty.StringVal("Hello World"),
		},
		{
			cty.StringVal(" "),
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("Hello"),
					cty.StringVal("World"),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("Foo"),
					cty.StringVal("Bar"),
				}),
			},
			cty.StringVal("Hello World Foo Bar"),
		},
		{
			cty.StringVal(" "),
			[]cty.Value{
				cty.ListValEmpty(cty.String),
			},
			cty.StringVal(""),
		},
		{
			cty.StringVal(" "),
			[]cty.Value{
				cty.ListValEmpty(cty.String),
				cty.ListValEmpty(cty.String),
				cty.ListValEmpty(cty.String),
			},
			cty.StringVal(""),
		},
		{
			cty.StringVal(" "),
			[]cty.Value{
				cty.ListValEmpty(cty.String),
				cty.ListVal([]cty.Value{
					cty.StringVal("Foo"),
					cty.StringVal("Bar"),
				}),
			},
			cty.StringVal("Foo Bar"),
		},
		{
			cty.UnknownVal(cty.String),
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("Hello"),
					cty.StringVal("World"),
				}),
			},
			cty.UnknownVal(cty.String),
		},
		{
			cty.StringVal(" "),
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.StringVal("Hello"),
					cty.UnknownVal(cty.String),
				}),
			},
			cty.UnknownVal(cty.String),
		},
		{
			cty.StringVal(" "),
			[]cty.Value{
				cty.UnknownVal(cty.List(cty.String)),
			},
			cty.UnknownVal(cty.String),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Join(%#v, %#v...)", test.Sep, test.Lists), func(t *testing.T) {
			got, err := Join(test.Sep, test.Lists...)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestSort(t *testing.T) {
	tests := []struct {
		List cty.Value
		Want cty.Value
	}{
		{
			cty.ListValEmpty(cty.String),
			cty.ListValEmpty(cty.String),
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("banana"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("banana"),
			}),
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("banana"),
				cty.StringVal("apple"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("apple"),
				cty.StringVal("banana"),
			}),
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("8"),
				cty.StringVal("9"),
				cty.StringVal("10"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal("10"), // lexicographical sort, not numeric sort
				cty.StringVal("8"),
				cty.StringVal("9"),
			}),
		},
		{
			cty.UnknownVal(cty.List(cty.String)),
			cty.UnknownVal(cty.List(cty.String)),
		},
		{
			cty.ListVal([]cty.Value{
				cty.UnknownVal(cty.String),
			}),
			cty.UnknownVal(cty.List(cty.String)),
		},
		{
			cty.ListVal([]cty.Value{
				cty.UnknownVal(cty.String),
				cty.StringVal("banana"),
			}),
			cty.UnknownVal(cty.List(cty.String)),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Sort(%#v)", test.List), func(t *testing.T) {
			got, err := Sort(test.List)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
func TestSplit(t *testing.T) {
	tests := []struct {
		Sep  cty.Value
		Str  cty.Value
		Want cty.Value
	}{
		{
			cty.StringVal(" "),
			cty.StringVal("Hello World"),
			cty.ListVal([]cty.Value{
				cty.StringVal("Hello"),
				cty.StringVal("World"),
			}),
		},
		{
			cty.StringVal(" "),
			cty.StringVal("Hello"),
			cty.ListVal([]cty.Value{
				cty.StringVal("Hello"),
			}),
		},
		{
			cty.StringVal(" "),
			cty.StringVal(""),
			cty.ListVal([]cty.Value{
				cty.StringVal(""),
			}),
		},
		{
			cty.StringVal(""),
			cty.StringVal(""),
			cty.ListValEmpty(cty.String),
		},
		{
			cty.UnknownVal(cty.String),
			cty.StringVal("Hello World"),
			cty.UnknownVal(cty.List(cty.String)),
		},
		{
			cty.StringVal(" "),
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.List(cty.String)),
		},
		{
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.List(cty.String)),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Split(%#v, %#v)", test.Sep, test.Str), func(t *testing.T) {
			got, err := Split(test.Sep, test.Str)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestChomp(t *testing.T) {
	tests := []struct {
		String cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("hello world"),
			cty.StringVal("hello world"),
			false,
		},
		{
			cty.StringVal("goodbye\ncruel\nworld"),
			cty.StringVal("goodbye\ncruel\nworld"),
			false,
		},
		{
			cty.StringVal("goodbye\r\nwindows\r\nworld"),
			cty.StringVal("goodbye\r\nwindows\r\nworld"),
			false,
		},
		{
			cty.StringVal("goodbye\ncruel\nworld\n"),
			cty.StringVal("goodbye\ncruel\nworld"),
			false,
		},
		{
			cty.StringVal("goodbye\ncruel\nworld\n\n\n\n"),
			cty.StringVal("goodbye\ncruel\nworld"),
			false,
		},
		{
			cty.StringVal("goodbye\r\nwindows\r\nworld\r\n"),
			cty.StringVal("goodbye\r\nwindows\r\nworld"),
			false,
		},
		{
			cty.StringVal("goodbye\r\nwindows\r\nworld\r\n\r\n\r\n\r\n"),
			cty.StringVal("goodbye\r\nwindows\r\nworld"),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("chomp(%#v)", test.String), func(t *testing.T) {
			got, err := Chomp(test.String)

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

func TestIndent(t *testing.T) {
	tests := []struct {
		String cty.Value
		Spaces cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal(`Fleas:
Adam
Had'em

E.E. Cummings`),
			cty.NumberIntVal(4),
			cty.StringVal("Fleas:\n    Adam\n    Had'em\n    \n    E.E. Cummings"),
			false,
		},
		{
			cty.StringVal("oneliner"),
			cty.NumberIntVal(4),
			cty.StringVal("oneliner"),
			false,
		},
		{
			cty.StringVal(`#!/usr/bin/env bash
date
pwd`),
			cty.NumberIntVal(4),
			cty.StringVal("#!/usr/bin/env bash\n    date\n    pwd"),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("indent(%#v, %#v)", test.Spaces, test.String), func(t *testing.T) {
			got, err := Indent(test.Spaces, test.String)

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

func TestReplace(t *testing.T) {
	tests := []struct {
		String  cty.Value
		Substr  cty.Value
		Replace cty.Value
		Want    cty.Value
		Err     bool
	}{
		{ // Regular search and replace
			cty.StringVal("hello"),
			cty.StringVal("hel"),
			cty.StringVal("bel"),
			cty.StringVal("bello"),
			false,
		},
		{ // Search string doesn't match
			cty.StringVal("hello"),
			cty.StringVal("nope"),
			cty.StringVal("bel"),
			cty.StringVal("hello"),
			false,
		},
		{ // Regular expression
			cty.StringVal("hello"),
			cty.StringVal("/l/"),
			cty.StringVal("L"),
			cty.StringVal("heLLo"),
			false,
		},
		{
			cty.StringVal("helo"),
			cty.StringVal("/(l)/"),
			cty.StringVal("$1$1"),
			cty.StringVal("hello"),
			false,
		},
		{ // Bad regexp
			cty.StringVal("hello"),
			cty.StringVal("/(l/"),
			cty.StringVal("$1$1"),
			cty.UnknownVal(cty.String),
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("replace(%#v, %#v, %#v)", test.String, test.Substr, test.Replace), func(t *testing.T) {
			got, err := Replace(test.String, test.Substr, test.Replace)

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

func TestTitle(t *testing.T) {
	tests := []struct {
		String cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal("hello"),
			cty.StringVal("Hello"),
			false,
		},
		{
			cty.StringVal("hello world"),
			cty.StringVal("Hello World"),
			false,
		},
		{
			cty.StringVal(""),
			cty.StringVal(""),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("title(%#v)", test.String), func(t *testing.T) {
			got, err := Title(test.String)

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

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		String cty.Value
		Want   cty.Value
		Err    bool
	}{
		{
			cty.StringVal(" hello "),
			cty.StringVal("hello"),
			false,
		},
		{
			cty.StringVal(""),
			cty.StringVal(""),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("trimspace(%#v)", test.String), func(t *testing.T) {
			got, err := TrimSpace(test.String)

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
