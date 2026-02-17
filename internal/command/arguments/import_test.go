// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func init() {
	// Mock getwd for tests to return empty string
	getwd = func() (string, error) {
		return "", nil
	}
}

func TestParseImport_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Import
	}{
		"defaults": {
			[]string{"test_instance.foo", "bar"},
			&Import{
				State:        &State{Lock: true},
				Vars:         &Vars{},
				ConfigPath:   "",
				Parallelism:  DefaultParallelism,
				InputEnabled: true,
				Addr:         "test_instance.foo",
				ID:           "bar",
			},
		},
		"state flag": {
			[]string{"-state", "mystate.tfstate", "test_instance.foo", "bar"},
			&Import{
				State:        &State{Lock: true, StatePath: "mystate.tfstate"},
				Vars:         &Vars{},
				Parallelism:  DefaultParallelism,
				InputEnabled: true,
				Addr:         "test_instance.foo",
				ID:           "bar",
			},
		},
		"state-out and backup flags": {
			[]string{"-state-out", "out.tfstate", "-backup", "backup.tfstate", "test_instance.foo", "bar"},
			&Import{
				State:        &State{Lock: true, StateOutPath: "out.tfstate", BackupPath: "backup.tfstate"},
				Vars:         &Vars{},
				Parallelism:  DefaultParallelism,
				InputEnabled: true,
				Addr:         "test_instance.foo",
				ID:           "bar",
			},
		},
		"lock disabled": {
			[]string{"-lock=false", "test_instance.foo", "bar"},
			&Import{
				State:        &State{Lock: false},
				Vars:         &Vars{},
				Parallelism:  DefaultParallelism,
				InputEnabled: true,
				Addr:         "test_instance.foo",
				ID:           "bar",
			},
		},
		"config path": {
			[]string{"-config=/tmp/config", "test_instance.foo", "bar"},
			&Import{
				State:        &State{Lock: true},
				Vars:         &Vars{},
				ConfigPath:   "/tmp/config",
				Parallelism:  DefaultParallelism,
				InputEnabled: true,
				Addr:         "test_instance.foo",
				ID:           "bar",
			},
		},
		"parallelism": {
			[]string{"-parallelism=5", "test_instance.foo", "bar"},
			&Import{
				State:        &State{Lock: true},
				Vars:         &Vars{},
				Parallelism:  5,
				InputEnabled: true,
				Addr:         "test_instance.foo",
				ID:           "bar",
			},
		},
		"ignore remote version": {
			[]string{"-ignore-remote-version", "test_instance.foo", "bar"},
			&Import{
				State:               &State{Lock: true},
				Vars:                &Vars{},
				Parallelism:         DefaultParallelism,
				IgnoreRemoteVersion: true,
				InputEnabled:        true,
				Addr:                "test_instance.foo",
				ID:                  "bar",
			},
		},
		"input disabled": {
			[]string{"-input=false", "test_instance.foo", "bar"},
			&Import{
				State:        &State{Lock: true},
				Vars:         &Vars{},
				Parallelism:  DefaultParallelism,
				InputEnabled: false,
				Addr:         "test_instance.foo",
				ID:           "bar",
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{}, State{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseImport(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Errorf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseImport_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Import
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop", "test_instance.foo", "bar"},
			&Import{
				State:        &State{Lock: true},
				Vars:         &Vars{},
				Parallelism:  DefaultParallelism,
				InputEnabled: true,
				Addr:         "test_instance.foo",
				ID:           "bar",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -boop",
				),
			},
		},
		"missing all arguments": {
			nil,
			&Import{
				State:        &State{Lock: true},
				Vars:         &Vars{},
				Parallelism:  DefaultParallelism,
				InputEnabled: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Wrong number of arguments",
					"The import command expects two arguments: ADDR and ID.",
				),
			},
		},
		"only one argument": {
			[]string{"test_instance.foo"},
			&Import{
				State:        &State{Lock: true},
				Vars:         &Vars{},
				Parallelism:  DefaultParallelism,
				InputEnabled: true,
				Addr:         "test_instance.foo",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Wrong number of arguments",
					"The import command expects two arguments: ADDR and ID.",
				),
			},
		},
		"too many arguments": {
			[]string{"test_instance.foo", "bar", "baz"},
			&Import{
				State:        &State{Lock: true},
				Vars:         &Vars{},
				Parallelism:  DefaultParallelism,
				InputEnabled: true,
				Addr:         "test_instance.foo",
				ID:           "bar",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Wrong number of arguments",
					"The import command expects two arguments: ADDR and ID.",
				),
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{}, State{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseImport(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Errorf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

func TestParseImport_vars(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want []FlagNameValue
	}{
		"no var flags by default": {
			args: []string{"test_instance.foo", "bar"},
			want: nil,
		},
		"one var": {
			args: []string{"-var", "foo=bar", "test_instance.foo", "bar"},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
			},
		},
		"one var-file": {
			args: []string{"-var-file", "cool.tfvars", "test_instance.foo", "bar"},
			want: []FlagNameValue{
				{Name: "-var-file", Value: "cool.tfvars"},
			},
		},
		"ordering preserved": {
			args: []string{
				"-var", "foo=bar",
				"-var-file", "cool.tfvars",
				"-var", "boop=beep",
				"test_instance.foo", "bar",
			},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
				{Name: "-var-file", Value: "cool.tfvars"},
				{Name: "-var", Value: "boop=beep"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseImport(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected result\n%s", cmp.Diff(vars, tc.want))
			}
		})
	}
}
