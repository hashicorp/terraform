// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseImport_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Import
	}{
		"defaults": {
			[]string{"test_instance.foo", "bar"},
			&Import{
				State:               &State{Lock: true},
				Vars:                &Vars{},
				ConfigPath:          "",
				InputEnabled:        true,
				Parallelism:         10,
				IgnoreRemoteVersion: false,
				Addr:                "test_instance.foo",
				ID:                  "bar",
			},
		},
		"with config path": {
			[]string{"-config=/tmp/config", "test_instance.foo", "bar"},
			&Import{
				State:               &State{Lock: true},
				Vars:                &Vars{},
				ConfigPath:          "/tmp/config",
				InputEnabled:        true,
				Parallelism:         10,
				IgnoreRemoteVersion: false,
				Addr:                "test_instance.foo",
				ID:                  "bar",
			},
		},
		"with state flags": {
			[]string{"-state=/tmp/state.tfstate", "-lock=false", "test_instance.foo", "bar"},
			&Import{
				State:               &State{StatePath: "/tmp/state.tfstate"},
				Vars:                &Vars{},
				ConfigPath:          "",
				InputEnabled:        true,
				Parallelism:         10,
				IgnoreRemoteVersion: false,
				Addr:                "test_instance.foo",
				ID:                  "bar",
			},
		},
		"ignore remote version": {
			[]string{"-ignore-remote-version", "test_instance.foo", "bar"},
			&Import{
				State:               &State{Lock: true},
				Vars:                &Vars{},
				ConfigPath:          "",
				InputEnabled:        true,
				Parallelism:         10,
				IgnoreRemoteVersion: true,
				Addr:                "test_instance.foo",
				ID:                  "bar",
			},
		},
		"parallelism": {
			[]string{"-parallelism=5", "test_instance.foo", "bar"},
			&Import{
				State:               &State{Lock: true},
				Vars:                &Vars{},
				ConfigPath:          "",
				InputEnabled:        true,
				Parallelism:         5,
				IgnoreRemoteVersion: false,
				Addr:                "test_instance.foo",
				ID:                  "bar",
			},
		},
		"input disabled": {
			[]string{"-input=false", "test_instance.foo", "bar"},
			&Import{
				State:               &State{Lock: true},
				Vars:                &Vars{},
				ConfigPath:          "",
				InputEnabled:        false,
				Parallelism:         10,
				IgnoreRemoteVersion: false,
				Addr:                "test_instance.foo",
				ID:                  "bar",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseImport(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			// Compare fields individually to avoid issues with Vars internal state
			if got.ConfigPath != tc.want.ConfigPath {
				t.Errorf("wrong ConfigPath\n got: %q\nwant: %q", got.ConfigPath, tc.want.ConfigPath)
			}
			if got.InputEnabled != tc.want.InputEnabled {
				t.Errorf("wrong InputEnabled\n got: %v\nwant: %v", got.InputEnabled, tc.want.InputEnabled)
			}
			if got.Parallelism != tc.want.Parallelism {
				t.Errorf("wrong Parallelism\n got: %d\nwant: %d", got.Parallelism, tc.want.Parallelism)
			}
			if got.IgnoreRemoteVersion != tc.want.IgnoreRemoteVersion {
				t.Errorf("wrong IgnoreRemoteVersion\n got: %v\nwant: %v", got.IgnoreRemoteVersion, tc.want.IgnoreRemoteVersion)
			}
			if got.Addr != tc.want.Addr {
				t.Errorf("wrong Addr\n got: %q\nwant: %q", got.Addr, tc.want.Addr)
			}
			if got.ID != tc.want.ID {
				t.Errorf("wrong ID\n got: %q\nwant: %q", got.ID, tc.want.ID)
			}
			if got.State.Lock != tc.want.State.Lock {
				t.Errorf("wrong State.Lock\n got: %v\nwant: %v", got.State.Lock, tc.want.State.Lock)
			}
			if got.State.StatePath != tc.want.State.StatePath {
				t.Errorf("wrong State.StatePath\n got: %q\nwant: %q", got.State.StatePath, tc.want.State.StatePath)
			}
		})
	}
}

func TestParseImport_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-unknown", "test_instance.foo", "bar"},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -unknown",
				),
			},
		},
		"too few arguments": {
			[]string{"test_instance.foo"},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid number of arguments",
					"The import command expects two arguments.",
				),
			},
		},
		"too many arguments": {
			[]string{"test_instance.foo", "bar", "baz"},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid number of arguments",
					"The import command expects two arguments.",
				),
			},
		},
		"no arguments": {
			nil,
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid number of arguments",
					"The import command expects two arguments.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			_, gotDiags := ParseImport(tc.args)
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}
