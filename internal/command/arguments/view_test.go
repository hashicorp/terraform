// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseView(t *testing.T) {
	testCases := map[string]struct {
		args     []string
		want     *View
		wantArgs []string
	}{
		"nil": {
			nil,
			&View{NoColor: false, CompactWarnings: false},
			nil,
		},
		"empty": {
			[]string{},
			&View{NoColor: false, CompactWarnings: false},
			[]string{},
		},
		"none matching": {
			[]string{"-foo", "bar", "-baz"},
			&View{NoColor: false, CompactWarnings: false},
			[]string{"-foo", "bar", "-baz"},
		},
		"no-color": {
			[]string{"-foo", "-no-color", "-baz"},
			&View{NoColor: true, CompactWarnings: false},
			[]string{"-foo", "-baz"},
		},
		"compact-warnings": {
			[]string{"-foo", "-compact-warnings", "-baz"},
			&View{NoColor: false, CompactWarnings: true},
			[]string{"-foo", "-baz"},
		},
		"both": {
			[]string{"-foo", "-no-color", "-compact-warnings", "-baz"},
			&View{NoColor: true, CompactWarnings: true},
			[]string{"-foo", "-baz"},
		},
		"both, resulting in empty args": {
			[]string{"-no-color", "-compact-warnings"},
			&View{NoColor: true, CompactWarnings: true},
			[]string{},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotArgs := ParseView(tc.args)
			if *got != *tc.want {
				t.Errorf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			if !cmp.Equal(gotArgs, tc.wantArgs) {
				t.Errorf("unexpected args\n got: %#v\nwant: %#v", gotArgs, tc.wantArgs)
			}
		})
	}
}
