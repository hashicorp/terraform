package hcl

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
			"assign_colon.hcl",
			true,
		},
		{
			"comment.hcl",
			false,
		},
		{
			"multiple.hcl",
			false,
		},
		{
			"structure.hcl",
			false,
		},
		{
			"structure_basic.hcl",
			false,
		},
		{
			"structure_empty.hcl",
			false,
		},
		{
			"complex.hcl",
			false,
		},
		{
			"assign_deep.hcl",
			true,
		},
		{
			"types.hcl",
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
