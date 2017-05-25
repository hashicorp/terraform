package json

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLexJson(t *testing.T) {
	cases := []struct {
		Input  string
		Output []int
	}{
		{
			"basic.json",
			[]int{
				LEFTBRACE,
				STRING, COLON, STRING,
				RIGHTBRACE,
				lexEOF,
			},
		},
		{
			"array.json",
			[]int{
				LEFTBRACE,
				STRING, COLON, LEFTBRACKET,
				NUMBER, COMMA, NUMBER, COMMA, STRING,
				RIGHTBRACKET, COMMA,
				STRING, COLON, STRING,
				RIGHTBRACE,
				lexEOF,
			},
		},
		{
			"object.json",
			[]int{
				LEFTBRACE,
				STRING, COLON, LEFTBRACE,
				STRING, COLON, LEFTBRACKET,
				NUMBER, COMMA, NUMBER,
				RIGHTBRACKET,
				RIGHTBRACE,
				RIGHTBRACE,
				lexEOF,
			},
		},
	}

	for _, tc := range cases {
		d, err := ioutil.ReadFile(filepath.Join(fixtureDir, tc.Input))
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		l := &jsonLex{Input: string(d)}
		var actual []int
		for {
			token := l.Lex(new(jsonSymType))
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
