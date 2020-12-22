package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
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

func TestSnakeCase(t *testing.T) {
	tests := []struct {
		Input cty.Value
		Want  cty.Value
	}{
		{
			cty.StringVal("hello_world"),
			cty.StringVal("hello_world"),
		},
		{
			cty.StringVal("HelloWorld"),
			cty.StringVal("hello_world"),
		},
		{
			cty.StringVal("ABC"),
			cty.StringVal("abc"),
		},
		{
			cty.StringVal("ABCd"),
			cty.StringVal("ab_cd"),
		},
		{
			cty.StringVal("_hello_world_"),
			cty.StringVal("hello_world"),
		},
		{
			cty.StringVal("snake_case"),
			cty.StringVal("snake_case"),
		},
		{
			cty.StringVal("PascalCase"),
			cty.StringVal("pascal_case"),
		},
		{
			cty.StringVal("camelCase"),
			cty.StringVal("camel_case"),
		},
		{
			cty.StringVal("kebab-case"),
			cty.StringVal("kebab_case"),
		},
		{
			cty.StringVal(""),
			cty.StringVal(""),
		},
		{
			cty.StringVal("1"),
			cty.StringVal("1"),
		},
	}

	for _, test := range tests {
		t.Run(test.Input.GoString(), func(t *testing.T) {
			got, err := SnakeCase(test.Input)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestKebabCase(t *testing.T) {
	tests := []struct {
		Input cty.Value
		Want  cty.Value
	}{
		{
			cty.StringVal("hello_world"),
			cty.StringVal("hello-world"),
		},
		{
			cty.StringVal("HelloWorld"),
			cty.StringVal("hello-world"),
		},
		{
			cty.StringVal("ABC"),
			cty.StringVal("abc"),
		},
		{
			cty.StringVal("ABCd"),
			cty.StringVal("ab-cd"),
		},
		{
			cty.StringVal("_hello_world_"),
			cty.StringVal("hello-world"),
		},
		{
			cty.StringVal("snake_case"),
			cty.StringVal("snake-case"),
		},
		{
			cty.StringVal("PascalCase"),
			cty.StringVal("pascal-case"),
		},
		{
			cty.StringVal("camelCase"),
			cty.StringVal("camel-case"),
		},
		{
			cty.StringVal("kebab-case"),
			cty.StringVal("kebab-case"),
		},
		{
			cty.StringVal(""),
			cty.StringVal(""),
		},
		{
			cty.StringVal("1"),
			cty.StringVal("1"),
		},
	}

	for _, test := range tests {
		t.Run(test.Input.GoString(), func(t *testing.T) {
			got, err := KebabCase(test.Input)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestCamelCase(t *testing.T) {
	tests := []struct {
		Input cty.Value
		Want  cty.Value
	}{
		{
			cty.StringVal("hello_world"),
			cty.StringVal("helloWorld"),
		},
		{
			cty.StringVal("HelloWorld"),
			cty.StringVal("helloWorld"),
		},
		{
			cty.StringVal("ABC"),
			cty.StringVal("abc"),
		},
		{
			cty.StringVal("ABCd"),
			cty.StringVal("abCd"),
		},
		{
			cty.StringVal("_hello_world_"),
			cty.StringVal("helloWorld"),
		},
		{
			cty.StringVal("snake_case"),
			cty.StringVal("snakeCase"),
		},
		{
			cty.StringVal("PascalCase"),
			cty.StringVal("pascalCase"),
		},
		{
			cty.StringVal("camelCase"),
			cty.StringVal("camelCase"),
		},
		{
			cty.StringVal("kebab-case"),
			cty.StringVal("kebabCase"),
		},
		{
			cty.StringVal(""),
			cty.StringVal(""),
		},
		{
			cty.StringVal("1"),
			cty.StringVal("1"),
		},
	}

	for _, test := range tests {
		t.Run(test.Input.GoString(), func(t *testing.T) {
			got, err := CamelCase(test.Input)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
