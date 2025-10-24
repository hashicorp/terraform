// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestParseInit_basicValid(t *testing.T) {
	var flagNameValue []FlagNameValue
	testCases := map[string]struct {
		args []string
		want *Init
	}{
		"with default options": {
			nil,
			&Init{
				FromModule:          "",
				Lockfile:            "",
				TestsDirectory:      "tests",
				ViewType:            ViewHuman,
				Backend:             true,
				Cloud:               true,
				Get:                 true,
				ForceInitCopy:       false,
				StateLock:           true,
				StateLockTimeout:    0,
				Reconfigure:         false,
				MigrateState:        false,
				Upgrade:             false,
				Json:                false,
				IgnoreRemoteVersion: false,
				BackendConfig: FlagNameValueSlice{
					FlagName: "-backend-config",
					Items:    &flagNameValue,
				},
				Vars:            &Vars{},
				InputEnabled:    true,
				CompactWarnings: false,
				TargetFlags:     nil,
			},
		},
		"setting multiple options": {
			[]string{"-backend=false", "-force-copy=true",
				"-from-module=./main-dir", "-json", "-get=false",
				"-lock=false", "-lock-timeout=10s", "-reconfigure=true",
				"-upgrade=true", "-lockfile=readonly", "-compact-warnings=true",
				"-ignore-remote-version=true", "-test-directory=./test-dir"},
			&Init{
				FromModule:          "./main-dir",
				Lockfile:            "readonly",
				TestsDirectory:      "./test-dir",
				ViewType:            ViewJSON,
				Backend:             false,
				Cloud:               false,
				Get:                 false,
				ForceInitCopy:       true,
				StateLock:           false,
				StateLockTimeout:    time.Duration(10) * time.Second,
				Reconfigure:         true,
				MigrateState:        false,
				Upgrade:             true,
				Json:                true,
				IgnoreRemoteVersion: true,
				BackendConfig: FlagNameValueSlice{
					FlagName: "-backend-config",
					Items:    &flagNameValue,
				},
				Vars:            &Vars{},
				InputEnabled:    true,
				Args:            []string{},
				CompactWarnings: true,
				TargetFlags:     nil,
			},
		},
		"with cloud option": {
			[]string{"-cloud=false", "-input=false", "-target=foo_bar.baz", "-backend-config", "backend.config"},
			&Init{
				FromModule:          "",
				Lockfile:            "",
				TestsDirectory:      "tests",
				ViewType:            ViewHuman,
				Backend:             false,
				Cloud:               false,
				Get:                 true,
				ForceInitCopy:       false,
				StateLock:           true,
				StateLockTimeout:    0,
				Reconfigure:         false,
				MigrateState:        false,
				Upgrade:             false,
				Json:                false,
				IgnoreRemoteVersion: false,
				BackendConfig: FlagNameValueSlice{
					FlagName: "-backend-config",
					Items:    &[]FlagNameValue{{Name: "-backend-config", Value: "backend.config"}},
				},
				Vars:            &Vars{},
				InputEnabled:    false,
				Args:            []string{},
				CompactWarnings: false,
				TargetFlags:     []string{"foo_bar.baz"},
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseInit(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}

			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Errorf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseInit_invalid(t *testing.T) {
	testCases := map[string]struct {
		args         []string
		wantErr      string
		wantViewType ViewType
	}{
		"with unsupported options": {
			args:         []string{"-raw"},
			wantErr:      "flag provided but not defined",
			wantViewType: ViewHuman,
		},
		"with both -backend and -cloud options set": {
			args:         []string{"-backend=false", "-cloud=false"},
			wantErr:      "The -backend and -cloud options are aliases of one another and mutually-exclusive in their use",
			wantViewType: ViewHuman,
		},
		"with both -migrate-state and -json options set": {
			args:         []string{"-migrate-state", "-json"},
			wantErr:      "Terraform cannot ask for interactive approval when -json is set. To use the -migrate-state option, disable the -json option.",
			wantViewType: ViewJSON,
		},
		"with both -migrate-state and -reconfigure options set": {
			args:         []string{"-migrate-state", "-reconfigure"},
			wantErr:      "The -migrate-state and -reconfigure options are mutually-exclusive.",
			wantViewType: ViewHuman,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseInit(tc.args)
			if len(diags) == 0 {
				t.Fatal("expected diags but got none")
			}
			if got, want := diags.Err().Error(), tc.wantErr; !strings.Contains(got, want) {
				t.Fatalf("wrong diags\n got: %s\nwant: %s", got, want)
			}
			if got.ViewType != tc.wantViewType {
				t.Fatalf("wrong view type, got %#v, want %#v", got.ViewType, ViewHuman)
			}
		})
	}
}

func TestParseInit_vars(t *testing.T) {
	testCases := map[string]struct {
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

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseInit(tc.args)
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
