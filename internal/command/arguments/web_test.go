package arguments

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/internal/command/webcommand"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseWeb_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Web
	}{
		"defaults": {
			nil,
			&Web{
				TargetObject: webcommand.TargetObjectCurrentWorkspace,
			},
		},
		"latest run": {
			[]string{"-latest-run"},
			&Web{
				TargetObject: webcommand.TargetObjectLatestRun,
			},
		},
		"specified run": {
			[]string{"-run=abc123"},
			&Web{
				TargetObject: webcommand.TargetObjectRun{RunID: "abc123"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseWeb(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseWeb_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Web
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop"},
			&Web{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -boop",
				),
			},
		},
		"both latest run and specified run": {
			[]string{"-latest-run", "-run=abc123"},
			&Web{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid combination of options",
					"Cannot use multiple object selection options in the same command.",
				),
			},
		},
		"specified run with no ID": {
			[]string{"-run"},
			&Web{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag needs an argument: -run",
				),
			},
		},
		"positional arguments": {
			[]string{"blah"},
			&Web{
				TargetObject: webcommand.TargetObjectCurrentWorkspace,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Unexpected argument",
					"The 'web' command does not expect any positional arguments.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseWeb(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			if !reflect.DeepEqual(gotDiags, tc.wantDiags) {
				t.Errorf("wrong result\ngot: %s\nwant: %s", spew.Sdump(gotDiags), spew.Sdump(tc.wantDiags))
			}
		})
	}
}
