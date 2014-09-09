package config

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLex(t *testing.T) {
	cases := []struct {
		Input  string
		Output []int
	}{
		{
			"concat.hcl",
			[]int{IDENTIFIER, LEFTPAREN,
				STRING, COMMA, STRING, COMMA, STRING,
				RIGHTPAREN, lexEOF},
		},
	}

	for _, tc := range cases {
		d, err := ioutil.ReadFile(filepath.Join(
			fixtureDir, "interpolations", tc.Input))
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		l := &exprLex{Input: string(d)}
		var actual []int
		for {
			token := l.Lex(new(exprSymType))
			actual = append(actual, token)

			if token == lexEOF {
				break
			}

			if len(actual) > 500 {
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
