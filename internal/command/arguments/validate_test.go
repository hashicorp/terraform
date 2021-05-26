package arguments

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseValidate_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Validate
	}{
		"defaults": {
			nil,
			&Validate{
				Path:     ".",
				ViewType: ViewHuman,
			},
		},
		"json": {
			[]string{"-json"},
			&Validate{
				Path:     ".",
				ViewType: ViewJSON,
			},
		},
		"path": {
			[]string{"-json", "foo"},
			&Validate{
				Path:     "foo",
				ViewType: ViewJSON,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseValidate(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseValidate_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Validate
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop"},
			&Validate{
				Path:     ".",
				ViewType: ViewHuman,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -boop",
				),
			},
		},
		"too many arguments": {
			[]string{"-json", "bar", "baz"},
			&Validate{
				Path:     "bar",
				ViewType: ViewJSON,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments",
					"Expected at most one positional argument.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseValidate(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			if !reflect.DeepEqual(gotDiags, tc.wantDiags) {
				t.Errorf("wrong result\ngot: %s\nwant: %s", spew.Sdump(gotDiags), spew.Sdump(tc.wantDiags))
			}
		})
	}
}
