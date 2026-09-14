// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseFmt_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Fmt
	}{
		"defaults": {
			want: &Fmt{
				List:  true,
				Write: true,
				Paths: []string{"."},
			},
		},
		"all options and paths": {
			args: []string{"-list=false", "-write=false", "-diff", "-check", "-recursive", "one.tf", "two.tf"},
			want: &Fmt{
				Diff:      true,
				Check:     true,
				Recursive: true,
				Paths:     []string{"one.tf", "two.tf"},
			},
		},
		"stdin": {
			args: []string{"-"},
			want: &Fmt{
				Paths: nil,
			},
		},
		"stdin with another target": {
			args: []string{"-", "other.tf"},
			want: &Fmt{
				Paths: nil,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseFmt(tc.args)
			if diags.HasErrors() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected parsed arguments\n%s", diff)
			}
		})
	}
}

func TestParseFmt_invalid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Fmt
	}{
		"unexpected flag": {
			args: []string{"-unknown"},
			want: &Fmt{List: true, Write: true, Paths: []string{"."}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseFmt(tc.args)
			if !diags.HasErrors() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected parsed arguments\n%s", diff)
			}
		})
	}
}
