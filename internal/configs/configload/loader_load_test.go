// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configload

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs"
)

func TestLoaderLoadConfig_okay(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/already-installed")
	loader, err := NewLoader(&Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	cfg, diags := loader.LoadConfig(fixtureDir)
	assertNoDiagnostics(t, diags)
	if cfg == nil {
		t.Fatalf("config is nil; want non-nil")
	}

	var gotPaths []string
	cfg.DeepEach(func(c *configs.Config) {
		gotPaths = append(gotPaths, strings.Join(c.Path, "."))
	})
	sort.Strings(gotPaths)
	wantPaths := []string{
		"", // root module
		"child_a",
		"child_a.child_c",
		"child_b",
		"child_b.child_d",
	}

	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("wrong module paths\ngot: %swant %s", spew.Sdump(gotPaths), spew.Sdump(wantPaths))
	}

	t.Run("child_a.child_c output", func(t *testing.T) {
		output := cfg.Children["child_a"].Children["child_c"].Module.Outputs["hello"]
		got, diags := output.Expr.Value(nil)
		assertNoDiagnostics(t, diags)
		assertResultCtyEqual(t, got, cty.StringVal("Hello from child_c"))
	})
	t.Run("child_b.child_d output", func(t *testing.T) {
		output := cfg.Children["child_b"].Children["child_d"].Module.Outputs["hello"]
		got, diags := output.Expr.Value(nil)
		assertNoDiagnostics(t, diags)
		assertResultCtyEqual(t, got, cty.StringVal("Hello from child_d"))
	})
}

func TestLoaderLoadConfig_addVersion(t *testing.T) {
	// This test is for what happens when there is a version constraint added
	// to a module that previously didn't have one.
	fixtureDir := filepath.Clean("testdata/add-version-constraint")
	loader, err := NewLoader(&Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	_, diags := loader.LoadConfig(fixtureDir)
	if !diags.HasErrors() {
		t.Fatalf("success; want error")
	}
	got := diags.Error()
	want := "Module version requirements have changed"
	if !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot:\n%s\n\nwant: containing %q", got, want)
	}
}

func TestLoaderLoadConfig_loadDiags(t *testing.T) {
	// building a config which didn't load correctly may cause configs to panic
	fixtureDir := filepath.Clean("testdata/invalid-names")
	loader, err := NewLoader(&Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	cfg, diags := loader.LoadConfig(fixtureDir)
	if !diags.HasErrors() {
		t.Fatal("success; want error")
	}

	if cfg == nil {
		t.Fatal("partial config not returned with diagnostics")
	}

	if cfg.Module == nil {
		t.Fatal("expected config module")
	}
}

func TestLoaderLoadConfig_loadDiagsFromSubmodules(t *testing.T) {
	// building a config which didn't load correctly may cause configs to panic
	fixtureDir := filepath.Clean("testdata/invalid-names-in-submodules")
	loader, err := NewLoader(&Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	cfg, diags := loader.LoadConfig(fixtureDir)
	if !diags.HasErrors() {
		t.Fatalf("loading succeeded; want an error")
	}
	if got, want := diags.Error(), " Invalid provider local name"; !strings.Contains(got, want) {
		t.Errorf("missing expected error\nwant substring: %s\ngot: %s", want, got)
	}

	if cfg == nil {
		t.Fatal("partial config not returned with diagnostics")
	}

	if cfg.Module == nil {
		t.Fatal("expected config module")
	}
}

func TestLoaderLoadConfig_childProviderGrandchildCount(t *testing.T) {
	// This test is focused on the specific situation where:
	// - A child module contains a nested provider block, which is no longer
	//   recommended but supported for backward-compatibility.
	// - A child of that child does _not_ contain a nested provider block,
	//   and is called with "count" (would also apply to "for_each" and
	//   "depends_on").
	// It isn't valid to use "count" with a module that _itself_ contains
	// a provider configuration, but it _is_ valid for a module with a
	// provider configuration to call another module with count. We previously
	// botched this rule and so this is a regression test to cover the
	// solution to that mistake:
	//     https://github.com/hashicorp/terraform/issues/31081

	// Since this test is based on success rather than failure and it's
	// covering a relatively large set of code where only a small part
	// contributes to the test, we'll make sure to test both the success and
	// failure cases here so that we'll have a better chance of noticing if a
	// future change makes this succeed only because we've reorganized the code
	// so that the check isn't happening at all anymore.
	//
	// If the "not okay" subtest fails, you should also be skeptical about
	// whether the "okay" subtest is still valid, even if it happens to
	// still be passing.
	t.Run("okay", func(t *testing.T) {
		fixtureDir := filepath.Clean("testdata/child-provider-grandchild-count")
		loader, err := NewLoader(&Config{
			ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
		})
		if err != nil {
			t.Fatalf("unexpected error from NewLoader: %s", err)
		}

		cfg, diags := loader.LoadConfig(fixtureDir)
		assertNoDiagnostics(t, diags)
		if cfg == nil {
			t.Fatalf("config is nil; want non-nil")
		}

		var gotPaths []string
		cfg.DeepEach(func(c *configs.Config) {
			gotPaths = append(gotPaths, strings.Join(c.Path, "."))
		})
		sort.Strings(gotPaths)
		wantPaths := []string{
			"", // root module
			"child",
			"child.grandchild",
		}

		if !reflect.DeepEqual(gotPaths, wantPaths) {
			t.Fatalf("wrong module paths\ngot: %swant %s", spew.Sdump(gotPaths), spew.Sdump(wantPaths))
		}
	})
	t.Run("not okay", func(t *testing.T) {
		fixtureDir := filepath.Clean("testdata/child-provider-child-count")
		loader, err := NewLoader(&Config{
			ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
		})
		if err != nil {
			t.Fatalf("unexpected error from NewLoader: %s", err)
		}

		_, diags := loader.LoadConfig(fixtureDir)
		if !diags.HasErrors() {
			t.Fatalf("loading succeeded; want an error")
		}
		if got, want := diags.Error(), "Module is incompatible with count, for_each, and depends_on"; !strings.Contains(got, want) {
			t.Errorf("missing expected error\nwant substring: %s\ngot: %s", want, got)
		}
	})

}
