// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package getproviders

import "testing"

func TestParsePlatform(t *testing.T) {
	tests := []struct {
		Input string
		Want  Platform
		Err   bool
	}{
		{
			"",
			Platform{},
			true,
		},
		{
			"too_many_notes",
			Platform{},
			true,
		},
		{
			"extra _ whitespaces ",
			Platform{},
			true,
		},
		{
			"arbitrary_os",
			Platform{OS: "arbitrary", Arch: "os"},
			false,
		},
	}

	for _, test := range tests {
		got, err := ParsePlatform(test.Input)
		if err != nil {
			if test.Err == false {
				t.Errorf("unexpected error: %s", err.Error())
			}
		} else {
			if test.Err {
				t.Errorf("wrong result: expected error, got none")
			}
		}
		if got != test.Want {
			t.Errorf("wrong\n got: %q\nwant: %q", got, test.Want)
		}
	}
}
