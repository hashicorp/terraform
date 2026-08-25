// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"flag"
	"testing"
)

func TestOptionName(t *testing.T) {
	testCases := map[string]struct {
		want   string
		wantOK bool
	}{
		"-dry-run":          {"dry-run", true},
		"--dry-run":         {"dry-run", true},
		"-id=bar":           {"id", true},
		"-j":                {"j", true},
		"module.eks":        {"", false},
		"test_instance.foo": {"", false},
		"hashicorp/google":  {"", false},
		"":                  {"", false},

		// A bare "-" means standard input to some commands, so it must not be
		// mistaken for an option.
		"-":  {"", false},
		"--": {"", false},

		// An invalid provider address may begin with a dash, but it does not
		// name an option.
		"-/-/google": {"/-/google", true},
	}

	for arg, tc := range testCases {
		t.Run(arg, func(t *testing.T) {
			got, gotOK := optionName(arg)
			if got != tc.want || gotOK != tc.wantOK {
				t.Errorf("optionName(%q) = %q, %v; want %q, %v", arg, got, gotOK, tc.want, tc.wantOK)
			}
		})
	}
}

func TestCheckStateMisplacedOptions(t *testing.T) {
	newFlagSet := func() *flag.FlagSet {
		var dryRun, lock bool
		fs := defaultFlagSet("state rm")
		fs.BoolVar(&dryRun, "dry-run", false, "dry run")
		fs.BoolVar(&lock, "lock", true, "lock state")
		return fs
	}

	testCases := map[string]struct {
		args      []string
		wantDiags int
	}{
		"only addresses": {
			[]string{"module.eks", "test_instance.foo"},
			0,
		},
		"misplaced option": {
			[]string{"module.eks", "-dry-run"},
			1,
		},
		"misplaced option with double dash": {
			[]string{"module.eks", "--dry-run"},
			1,
		},
		"misplaced option with value": {
			[]string{"module.eks", "-lock=false"},
			1,
		},
		// Every misplaced option is reported, so that a user who moved several
		// of them after the arguments learns about all of them at once.
		"several misplaced options": {
			[]string{"module.eks", "-dry-run", "-lock=false"},
			2,
		},
		// An argument that begins with a dash but does not name an option this
		// command defines is left alone, so that the command can report it as
		// the invalid address it is.
		"dashed argument that is not an option": {
			[]string{"-/-/google", "acmecorp/google"},
			0,
		},
		"bare dash": {
			[]string{"module.eks", "-"},
			0,
		},
		"unknown option": {
			[]string{"module.eks", "-boop"},
			0,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			diags := checkStateMisplacedOptions(newFlagSet(), "state rm", "ADDRESS...", tc.args)
			if got := len(diags); got != tc.wantDiags {
				t.Fatalf("got %d diagnostics, want %d: %s", got, tc.wantDiags, diags.Err())
			}
		})
	}
}
