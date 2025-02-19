// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseTest_Vars(t *testing.T) {
	tcs := map[string]struct {
		args []string
		want []FlagNameValue
	}{
		"no var flags by default": {
			args: nil,
			want: nil,
		},
		"one var": {
			args: []string{"-var", "foo=bar"},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
			},
		},
		"one var-file": {
			args: []string{"-var-file", "cool.tfvars"},
			want: []FlagNameValue{
				{Name: "-var-file", Value: "cool.tfvars"},
			},
		},
		"ordering preserved": {
			args: []string{
				"-var", "foo=bar",
				"-var-file", "cool.tfvars",
				"-var", "boop=beep",
			},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
				{Name: "-var-file", Value: "cool.tfvars"},
				{Name: "-var", Value: "boop=beep"},
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseTest(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected result\n%s", cmp.Diff(vars, tc.want))
			}
			if got, want := got.Vars.Empty(), len(tc.want) == 0; got != want {
				t.Fatalf("expected Empty() to return %t, but was %t", want, got)
			}
		})
	}
}

func TestParseTest(t *testing.T) {
	tcs := map[string]struct {
		args      []string
		want      *Test
		wantDiags tfdiags.Diagnostics
	}{
		"defaults": {
			args: nil,
			want: &Test{
				Filter:               nil,
				TestDirectory:        "tests",
				ViewType:             ViewHuman,
				Vars:                 &Vars{},
				OperationParallelism: 10,
			},
			wantDiags: nil,
		},
		"with-filters": {
			args: []string{"-filter=one.tftest.hcl", "-filter=two.tftest.hcl"},
			want: &Test{
				Filter:               []string{"one.tftest.hcl", "two.tftest.hcl"},
				TestDirectory:        "tests",
				ViewType:             ViewHuman,
				Vars:                 &Vars{},
				OperationParallelism: 10,
			},
			wantDiags: nil,
		},
		"json": {
			args: []string{"-json"},
			want: &Test{
				Filter:               nil,
				TestDirectory:        "tests",
				ViewType:             ViewJSON,
				Vars:                 &Vars{},
				OperationParallelism: 10,
			},
			wantDiags: nil,
		},
		"test-directory": {
			args: []string{"-test-directory=other"},
			want: &Test{
				Filter:               nil,
				TestDirectory:        "other",
				ViewType:             ViewHuman,
				Vars:                 &Vars{},
				OperationParallelism: 10,
			},
			wantDiags: nil,
		},
		"verbose": {
			args: []string{"-verbose"},
			want: &Test{
				Filter:               nil,
				TestDirectory:        "tests",
				ViewType:             ViewHuman,
				Verbose:              true,
				Vars:                 &Vars{},
				OperationParallelism: 10,
			},
		},
		"with-parallelism-set": {
			args: []string{"-parallelism=5"},
			want: &Test{
				Filter:               nil,
				TestDirectory:        "tests",
				ViewType:             ViewHuman,
				Vars:                 &Vars{},
				OperationParallelism: 5,
			},
			wantDiags: nil,
		},
		"with-parallelism-0": {
			args: []string{"-parallelism=0"},
			want: &Test{
				Filter:               nil,
				TestDirectory:        "tests",
				ViewType:             ViewHuman,
				Vars:                 &Vars{},
				OperationParallelism: 10,
			},
			wantDiags: nil,
		},
		"cloud-with-parallelism-0": {
			args: []string{"-parallelism=0", "-cloud-run=foobar"},
			want: &Test{
				CloudRunSource:       "foobar",
				Filter:               nil,
				TestDirectory:        "tests",
				ViewType:             ViewHuman,
				Vars:                 &Vars{},
				OperationParallelism: 0,
			},
			wantDiags: nil,
		},
		"unknown flag": {
			args: []string{"-boop"},
			want: &Test{
				Filter:               nil,
				TestDirectory:        "tests",
				ViewType:             ViewHuman,
				Vars:                 &Vars{},
				OperationParallelism: 10,
			},
			wantDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -boop",
				),
			},
		},
		"incompatible flags: -junit-xml and -cloud-run": {
			args: []string{"-junit-xml=./output.xml", "-cloud-run=foobar"},
			want: &Test{
				CloudRunSource:       "foobar",
				JUnitXMLFile:         "./output.xml",
				Filter:               nil,
				TestDirectory:        "tests",
				ViewType:             ViewHuman,
				Vars:                 &Vars{},
				OperationParallelism: 10,
			},
			wantDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Incompatible command-line flags",
					"The -junit-xml option is currently not compatible with remote test execution via the -cloud-run flag. If you are interested in JUnit XML output for remotely-executed tests please open an issue in GitHub.",
				),
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Operation{}, Vars{}, State{})

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseTest(tc.args)

			if diff := cmp.Diff(tc.want, got, cmpOpts); len(diff) > 0 {
				t.Errorf("diff:\n%s", diff)
			}

			tfdiags.AssertDiagnosticsMatch(t, diags, tc.wantDiags)
		})
	}
}
