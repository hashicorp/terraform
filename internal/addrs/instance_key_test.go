// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import (
	"fmt"
	"testing"
)

func TestInstanceKeyString(t *testing.T) {
	tests := []struct {
		Key  InstanceKey
		Want string
	}{
		{
			IntKey(0),
			`[0]`,
		},
		{
			IntKey(5),
			`[5]`,
		},
		{
			StringKey(""),
			`[""]`,
		},
		{
			StringKey("hi"),
			`["hi"]`,
		},
		{
			StringKey("0"),
			`["0"]`, // intentionally distinct from IntKey(0)
		},
		{
			// Quotes must be escaped
			StringKey(`"`),
			`["\""]`,
		},
		{
			// Escape sequences must themselves be escaped
			StringKey(`\r\n`),
			`["\\r\\n"]`,
		},
		{
			// Template interpolation sequences "${" must be escaped.
			StringKey(`${hello}`),
			`["$${hello}"]`,
		},
		{
			// Template control sequences "%{" must be escaped.
			StringKey(`%{ for something in something }%{ endfor }`),
			`["%%{ for something in something }%%{ endfor }"]`,
		},
		{
			// Dollar signs that aren't followed by { are not interpolation sequences
			StringKey(`$hello`),
			`["$hello"]`,
		},
		{
			// Percent signs that aren't followed by { are not control sequences
			StringKey(`%hello`),
			`["%hello"]`,
		},
	}

	for _, test := range tests {
		testName := fmt.Sprintf("%#v", test.Key)
		t.Run(testName, func(t *testing.T) {
			got := test.Key.String()
			want := test.Want
			if got != want {
				t.Errorf("wrong result\nreciever: %s\ngot:      %s\nwant:     %s", testName, got, want)
			}
		})
	}
}
