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
			"foo $${bar}",
			[]int{STRING, lexEOF},
		},

		{
			"foo $$$${bar}",
			[]int{STRING, lexEOF},
		},

		{
			"foo ${\"bar\"}",
			[]int{STRING, PROGRAM_BRACKET_LEFT, STRING, PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			"${bar(baz)}",
			[]int{PROGRAM_BRACKET_LEFT,
				IDENTIFIER, PAREN_LEFT, IDENTIFIER, PAREN_RIGHT,
				PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			"${bar(baz, foo)}",
			[]int{PROGRAM_BRACKET_LEFT,
				IDENTIFIER, PAREN_LEFT,
				IDENTIFIER, COMMA, IDENTIFIER,
				PAREN_RIGHT,
				PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			"${bar(42)}",
			[]int{PROGRAM_BRACKET_LEFT,
				IDENTIFIER, PAREN_LEFT, INTEGER, PAREN_RIGHT,
				PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			"${bar(-42)}",
			[]int{PROGRAM_BRACKET_LEFT,
				IDENTIFIER, PAREN_LEFT, ARITH_OP, INTEGER, PAREN_RIGHT,
				PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			"${bar(-42.0)}",
			[]int{PROGRAM_BRACKET_LEFT,
				IDENTIFIER, PAREN_LEFT, ARITH_OP, FLOAT, PAREN_RIGHT,
				PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			"${bar(42+1)}",
			[]int{PROGRAM_BRACKET_LEFT,
				IDENTIFIER, PAREN_LEFT,
				INTEGER, ARITH_OP, INTEGER,
				PAREN_RIGHT,
				PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			"${bar(42+-1)}",
			[]int{PROGRAM_BRACKET_LEFT,
				IDENTIFIER, PAREN_LEFT,
				INTEGER, ARITH_OP, ARITH_OP, INTEGER,
				PAREN_RIGHT,
				PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			"${bar(3.14159)}",
			[]int{PROGRAM_BRACKET_LEFT,
				IDENTIFIER, PAREN_LEFT, FLOAT, PAREN_RIGHT,
				PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			"${bar(inner(baz))}",
			[]int{PROGRAM_BRACKET_LEFT,
				IDENTIFIER, PAREN_LEFT,
				IDENTIFIER, PAREN_LEFT,
				IDENTIFIER,
				PAREN_RIGHT, PAREN_RIGHT,
				PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			"foo ${foo.bar.baz}",
			[]int{STRING, PROGRAM_BRACKET_LEFT, IDENTIFIER, PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			"foo ${foo.bar.*.baz}",
			[]int{STRING, PROGRAM_BRACKET_LEFT, IDENTIFIER, PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			"foo ${foo(\"baz\")}",
			[]int{STRING, PROGRAM_BRACKET_LEFT,
				IDENTIFIER, PAREN_LEFT, STRING, PAREN_RIGHT,
				PROGRAM_BRACKET_RIGHT, lexEOF},
		},

		{
			`foo ${"${var.foo}"}`,
			[]int{STRING, PROGRAM_BRACKET_LEFT,
				PROGRAM_BRACKET_LEFT, IDENTIFIER, PROGRAM_BRACKET_RIGHT,
				PROGRAM_BRACKET_RIGHT, lexEOF},
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
