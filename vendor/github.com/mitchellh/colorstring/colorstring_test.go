package colorstring

import (
	"os"
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

func TestColorPrefix(t *testing.T) {
	cases := []struct {
		Input, Output string
	}{
		{
			Input:  "foo",
			Output: "",
		},

		{
			Input:  "[blue]foo",
			Output: "[blue]",
		},

		{
			Input:  "[bold][blue]foo",
			Output: "[bold][blue]",
		},

		{
			Input:  "   [bold][blue]foo",
			Output: "[bold][blue]",
		},
	}

	for _, tc := range cases {
		actual := ColorPrefix(tc.Input)
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

	cases := []struct {
		Input, Output string
	}{
		{
			"[blue]foo",
			"foo",
		},

		{
			"[foo]bar",
			"[foo]bar",
		},
	}

	for _, tc := range cases {
		actual := c.Color(tc.Input)
		if actual != tc.Output {
			t.Errorf(
				"Input: %#v\n\nOutput: %#v\n\nExpected: %#v",
				tc.Input,
				actual,
				tc.Output)
		}
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

func TestConvenienceWrappers(t *testing.T) {
	var length int
	printInput := "[bold]Print:\t\t[default][red]R[green]G[blue]B[cyan]C[magenta]M[yellow]Y\n"
	printlnInput := "[bold]Println:\t[default][red]R[green]G[blue]B[cyan]C[magenta]M[yellow]Y"
	printfInput := "[bold]Printf:\t\t[default][red]R[green]G[blue]B[cyan]C[magenta]M[yellow]Y\n"
	fprintInput := "[bold]Fprint:\t\t[default][red]R[green]G[blue]B[cyan]C[magenta]M[yellow]Y\n"
	fprintlnInput := "[bold]Fprintln:\t[default][red]R[green]G[blue]B[cyan]C[magenta]M[yellow]Y"
	fprintfInput := "[bold]Fprintf:\t[default][red]R[green]G[blue]B[cyan]C[magenta]M[yellow]Y\n"

	// colorstring.Print
	length, _ = Print(printInput)
	assertOutputLength(t, printInput, 58, length)

	// colorstring.Println
	length, _ = Println(printlnInput)
	assertOutputLength(t, printlnInput, 59, length)

	// colorstring.Printf
	length, _ = Printf(printfInput)
	assertOutputLength(t, printfInput, 59, length)

	// colorstring.Fprint
	length, _ = Fprint(os.Stdout, fprintInput)
	assertOutputLength(t, fprintInput, 59, length)

	// colorstring.Fprintln
	length, _ = Fprintln(os.Stdout, fprintlnInput)
	assertOutputLength(t, fprintlnInput, 60, length)

	// colorstring.Fprintf
	length, _ = Fprintf(os.Stdout, fprintfInput)
	assertOutputLength(t, fprintfInput, 59, length)
}

func assertOutputLength(t *testing.T, input string, expectedLength int, actualLength int) {
	if actualLength != expectedLength {
		t.Errorf("Input: %#v\n\n Output length: %d\n\n Expected: %d",
			input,
			actualLength,
			expectedLength)
	}
}
