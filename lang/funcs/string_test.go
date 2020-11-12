package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

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

func TestTemplate(t *testing.T) {
	tests := []struct {
		String cty.Value
		Vars   cty.Value
		Want   cty.Value
		Err    string
	}{
		{
			cty.StringVal("Hello, ${name}!"),
			cty.EmptyObjectVal,
			cty.NilVal,
			`vars map does not contain key "name", referenced at str:1,10-14`,
		},
		{
			cty.StringVal("\xDF"),
			cty.EmptyObjectVal,
			cty.NilVal,
			`str:1,1-2: Invalid character encoding; All input files must be UTF-8 encoded. Ensure that UTF-8 encoding is selected in your editor., and 1 other diagnostic(s)`,
		},
		{
			cty.StringVal(""),
			cty.MapVal(map[string]cty.Value{
				"name": cty.StringVal("Jodie"),
			}),
			cty.StringVal(""),
			``,
		},
		{
			cty.NilVal,
			cty.EmptyObjectVal,
			cty.NilVal,
			`argument must not be null`,
		},
		{
			cty.StringVal("Hello, ${name}!"),
			cty.MapVal(map[string]cty.Value{
				"name": cty.StringVal("Jodie"),
			}),
			cty.StringVal("Hello, Jodie!"),
			``,
		},
		{
			cty.StringVal("Hello, ${name}!"),
			cty.MapVal(map[string]cty.Value{
				"name!": cty.StringVal("Jodie"),
			}),
			cty.NilVal,
			`invalid template variable name "name!": must start with a letter, followed by zero or more letters, digits, and underscores`,
		},
		{
			cty.StringVal("Hello, ${name}!"),
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("Jimbo"),
			}),
			cty.StringVal("Hello, Jimbo!"),
			``,
		},
		{
			cty.StringVal("The items are ${join(\", \", list)}"),
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
			cty.StringVal("Hello, ${template(\"\",{})}!"),
			cty.MapValEmpty(cty.String),
			cty.NilVal,
			`str:1,10-19: Error in function call; Call to function "template" failed: cannot recursively call template from inside template call.`,
		},
		{
			cty.StringVal("%{ for x in list ~}\n- ${x}\n%{ endfor ~}"),
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
			cty.StringVal("%{ for x in list ~}\n- ${x}\n%{ endfor ~}"),
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.True,
			}),
			cty.NilVal,
			`str:1,13-17: Iteration over non-iterable value; A value of type bool cannot be used as the collection in a 'for' expression.`,
		},
		{
			cty.StringVal("${val}"),
			cty.ObjectVal(map[string]cty.Value{
				"val": cty.True,
			}),
			cty.True, // since this template contains only an interpolation, its true value shines through
			``,
		},
	}

	templateFn := MakeTemplateFunc(func() map[string]function.Function {
		return map[string]function.Function{
			"join":     stdlib.JoinFunc,
			"template": stdlib.JoinFunc, // just a placeholder, since template itself overrides this
		}
	})

	for _, test := range tests {
		t.Run(fmt.Sprintf("Template(%#v, %#v)", test.String, test.Vars), func(t *testing.T) {
			got, err := templateFn.Call([]cty.Value{test.String, test.Vars})

			if argErr, ok := err.(function.ArgError); ok {
				if argErr.Index < 0 || argErr.Index > 1 {
					t.Errorf("ArgError index %d is out of range for template (must be 0 or 1)", argErr.Index)
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
