package arguments

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseOutput_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Output
	}{
		"defaults": {
			nil,
			&Output{
				Name:      "",
				ViewType:  ViewHuman,
				StatePath: "",
			},
		},
		"json": {
			[]string{"-json"},
			&Output{
				Name:      "",
				ViewType:  ViewJSON,
				StatePath: "",
			},
		},
		"raw": {
			[]string{"-raw", "foo"},
			&Output{
				Name:      "foo",
				ViewType:  ViewRaw,
				StatePath: "",
			},
		},
		"state": {
			[]string{"-state=foobar.tfstate", "-raw", "foo"},
			&Output{
				Name:      "foo",
				ViewType:  ViewRaw,
				StatePath: "foobar.tfstate",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseOutput(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseOutput_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Output
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop"},
			&Output{
				Name:      "",
				ViewType:  ViewHuman,
				StatePath: "",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -boop",
				),
			},
		},
		"json and raw specified": {
			[]string{"-json", "-raw"},
			&Output{
				Name:      "",
				ViewType:  ViewHuman,
				StatePath: "",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid output format",
					"The -raw and -json options are mutually-exclusive.",
				),
			},
		},
		"raw with no name": {
			[]string{"-raw"},
			&Output{
				Name:      "",
				ViewType:  ViewRaw,
				StatePath: "",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Output name required",
					"You must give the name of a single output value when using the -raw option.",
				),
			},
		},
		"too many arguments": {
			[]string{"-raw", "-state=foo.tfstate", "bar", "baz"},
			&Output{
				Name:      "bar",
				ViewType:  ViewRaw,
				StatePath: "foo.tfstate",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Unexpected argument",
					"The output command expects exactly one argument with the name of an output variable or no arguments to show all outputs.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseOutput(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			if !reflect.DeepEqual(gotDiags, tc.wantDiags) {
				t.Errorf("wrong result\ngot: %s\nwant: %s", spew.Sdump(gotDiags), spew.Sdump(tc.wantDiags))
			}
		})
	}
}
