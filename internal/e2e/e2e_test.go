// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package e2e

import (
	"strings"
	"testing"
)

// TestIsolateLocalProviderEnv asserts that IsolateLocalProviderEnv
// installs the specific environment-variable overrides we rely on to
// neutralise host-machine state during e2e tests. The set is derived
// from cliconfig.ConfigDir's HOME usage and from go-userdirs's
// per-platform search paths; if either of those grows a new variable
// in future, this test pins the contract so callers know to update
// the helper.
//
// We don't shell out here — the goal is just to confirm the helper
// touches the env table the way Cmd() will later read it. The
// integration of the helper with TestInitProvidersLocalOnly itself
// is the load-bearing acceptance test (a hostile HOME with a
// conflicting plugin in ~/.terraform.d/plugins makes that test fail
// on origin/main and pass on this branch — see GH-37501).
func TestIsolateLocalProviderEnv(t *testing.T) {
	b := &binary{}
	got := b.IsolateLocalProviderEnv(t)
	if got == "" {
		t.Fatalf("IsolateLocalProviderEnv returned empty path")
	}

	want := []string{
		"HOME=",
		"XDG_DATA_HOME=",
		"XDG_DATA_DIRS=",
		"XDG_CONFIG_HOME=",
		"XDG_CONFIG_DIRS=",
		"XDG_CACHE_HOME=",
		"APPDATA=",
		"LOCALAPPDATA=",
	}
	for _, prefix := range want {
		found := false
		for _, e := range b.env {
			if strings.HasPrefix(e, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected env entry with prefix %q to be set; have: %v", prefix, b.env)
		}
	}

	// HOME, APPDATA, and LOCALAPPDATA must point at the returned
	// isolated tmpdir so that any code path which derives further
	// directories from them lands inside the tmpdir rather than at
	// the real user home.
	for _, name := range []string{"HOME", "APPDATA", "LOCALAPPDATA"} {
		want := name + "=" + got
		found := false
		for _, e := range b.env {
			if e == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q in env, got entries: %v", want, b.env)
		}
	}

	// XDG_* must be set to empty strings so the unix backend of
	// go-userdirs falls back to spec defaults under the isolated
	// HOME (see go-userdirs/userdirs/app_unix_test.go for the same
	// no-XDG-set scenario).
	for _, name := range []string{
		"XDG_DATA_HOME",
		"XDG_DATA_DIRS",
		"XDG_CONFIG_HOME",
		"XDG_CONFIG_DIRS",
		"XDG_CACHE_HOME",
	} {
		want := name + "="
		found := false
		for _, e := range b.env {
			if e == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q (empty) in env, got entries: %v", want, b.env)
		}
	}
}
