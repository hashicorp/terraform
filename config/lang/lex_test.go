package lang

import (
	"reflect"
	"testing"
)

func TestLex(t *testing.T) {
	cases := []struct {
		Input  string
		Output []int
	}{
		{
			"foo",
			[]int{STRING, lexEOF},
		},

		{
			"foo$bar",
			[]int{STRING, lexEOF},
		},

		{
			"foo ${bar}",
			[]int{STRING, PROGRAM_BRACKET_LEFT, IDENTIFIER, PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			"foo ${\"bar\"}",
			[]int{STRING, PROGRAM_BRACKET_LEFT, STRING, PROGRAM_BRACKET_RIGHT, lexEOF},
		},
	}

	for _, tc := range cases {
		l := &parserLex{Input: tc.Input}
		var actual []int
		for {
			token := l.Lex(new(parserSymType))
			actual = append(actual, token)

			if token == lexEOF {
				break
			}

			// Be careful against what are probably infinite loops
			if len(actual) > 100 {
				t.Fatalf("Input:%s\n\nExausted.", tc.Input)
			}
		}

		if !reflect.DeepEqual(actual, tc.Output) {
			t.Fatalf(
				"Input: %s\n\nBad: %#v\n\nExpected: %#v",
				tc.Input, actual, tc.Output)
		}
	}
}

/* OTHERS:

foo ${var.foo}
bar ${"hello"}
foo ${concat("foo ${var.bar}", var.baz)}

*/
