package colorstring

import (
	"testing"
)

func TestColor(t *testing.T) {
	cases := []struct {
		Input, Output string
	}{
		{
			Input:  "foo",
			Output: "foo",
		},

		{
			Input:  "[blue]foo",
			Output: "\033[34mfoo\033[0m",
		},

		{
			Input:  "foo[blue]foo",
			Output: "foo\033[34mfoo\033[0m",
		},

		{
			Input:  "foo[what]foo",
			Output: "foo[what]foo",
		},
		{
			Input:  "foo[_blue_]foo",
			Output: "foo\033[44mfoo\033[0m",
		},
		{
			Input:  "foo[bold]foo",
			Output: "foo\033[1mfoo\033[0m",
		},
		{
			Input:  "[blue]foo[bold]bar",
			Output: "\033[34mfoo\033[1mbar\033[0m",
		},
		{
			Input:  "[underline]foo[reset]bar",
			Output: "\033[4mfoo\033[0mbar\033[0m",
		},
	}

	for _, tc := range cases {
		actual := Color(tc.Input)
		if actual != tc.Output {
			t.Errorf(
				"Input: %#v\n\nOutput: %#v\n\nExpected: %#v",
				tc.Input,
				actual,
				tc.Output)
		}
	}
}

func TestColorizeColor_disable(t *testing.T) {
	c := def
	c.Disable = true

	input := "[blue]foo"
	output := "foo"
	actual := c.Color(input)
	if actual != output {
		t.Errorf(
			"Input: %#v\n\nOutput: %#v\n\nExpected: %#v",
			input,
			actual,
			output)
	}
}

func TestColorizeColor_noReset(t *testing.T) {
	c := def
	c.Reset = false

	input := "[blue]foo"
	output := "\033[34mfoo"
	actual := c.Color(input)
	if actual != output {
		t.Errorf(
			"Input: %#v\n\nOutput: %#v\n\nExpected: %#v",
			input,
			actual,
			output)
	}
}
