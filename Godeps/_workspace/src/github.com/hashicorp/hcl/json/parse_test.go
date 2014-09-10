package json

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		Name string
		Err  bool
	}{
		{
			"basic.json",
			false,
		},
		{
			"object.json",
			false,
		},
		{
			"array.json",
			false,
		},
		{
			"types.json",
			false,
		},
	}

	for _, tc := range cases {
		d, err := ioutil.ReadFile(filepath.Join(fixtureDir, tc.Name))
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		_, err = Parse(string(d))
		if (err != nil) != tc.Err {
			t.Fatalf("Input: %s\n\nError: %s", tc.Name, err)
		}
	}
}
