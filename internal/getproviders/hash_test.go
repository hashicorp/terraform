// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package getproviders

import (
	"testing"
)

func TestParseHash(t *testing.T) {
	tests := []struct {
		Input   string
		Want    Hash
		WantErr string
	}{
		{
			Input: "h1:foo",
			Want:  HashScheme1.New("foo"),
		},
		{
			Input: "zh:bar",
			Want:  HashSchemeZip.New("bar"),
		},
		{
			// A scheme we don't know is considered valid syntax, it just won't match anything.
			Input: "unknown:baz",
			Want:  HashScheme("unknown:").New("baz"),
		},
		{
			// A scheme with an empty value is weird, but allowed.
			Input: "unknown:",
			Want:  HashScheme("unknown:").New(""),
		},
		{
			Input:   "",
			WantErr: "hash string must start with a scheme keyword followed by a colon",
		},
		{
			// A naked SHA256 hash in hex format is not sufficient
			Input:   "1e5f7a5f3ade7b8b1d1d59c5cea2e1a2f8d2f8c3f41962dbbe8647e222be8239",
			WantErr: "hash string must start with a scheme keyword followed by a colon",
		},
		{
			// An empty scheme is not allowed
			Input:   ":blah",
			WantErr: "hash string must start with a scheme keyword followed by a colon",
		},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			got, err := ParseHash(test.Input)

			if test.WantErr != "" {
				if err == nil {
					t.Fatalf("want error: %s", test.WantErr)
				}
				if got, want := err.Error(), test.WantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if got != test.Want {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
