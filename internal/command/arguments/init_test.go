// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"flag"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestParseInit_basicValid(t *testing.T) {
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
			},
		},
		"setting multiple options": {
			[]string{"-backend=false", "-force-copy=true",
				"-from-module=./main-dir", "-json", "-get=false",
				"-lock=false", "-lock-timeout=10s", "-reconfigure=true",
				"-upgrade=true", "-lockfile=readonly",
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
			},
		},
		"with cloud option": {
			[]string{"-cloud=false"},
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
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			cmdFlags := flag.NewFlagSet("init", flag.ContinueOnError)
			cmdFlags.SetOutput(io.Discard)

			got, diags := ParseInit(tc.args, cmdFlags)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseInit_invalid(t *testing.T) {
	testCases := map[string]struct {
		args    []string
		wantErr string
	}{
		"with unsupported options": {
			args:    []string{"-raw"},
			wantErr: "flag provided but not defined",
		},
		"with both -backend and -cloud options set": {
			args:    []string{"-backend=false", "-cloud=false"},
			wantErr: "The -backend and -cloud options are aliases of one another and mutually-exclusive in their use",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			cmdFlags := flag.NewFlagSet("init", flag.ContinueOnError)
			cmdFlags.SetOutput(io.Discard)

			got, diags := ParseInit(tc.args, cmdFlags)
			if len(diags) == 0 {
				t.Fatal("expected diags but got none")
			}
			if got, want := diags.Err().Error(), tc.wantErr; !strings.Contains(got, want) {
				t.Fatalf("wrong diags\n got: %s\nwant: %s", got, want)
			}
			if got.ViewType != ViewHuman {
				t.Fatalf("wrong view type, got %#v, want %#v", got.ViewType, ViewHuman)
			}
		})
	}
}
