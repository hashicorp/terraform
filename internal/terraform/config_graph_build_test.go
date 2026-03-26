// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-test/deep"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestLoadConfigWithSnapshot(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/config-graph/already-installed")
	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	_, got, diags := testLoadWithSnapshot(fixtureDir, loader, nil)
	assertNoDiagnostics(t, diags)
	if got == nil {
		t.Fatalf("snapshot is nil; want non-nil")
	}

	t.Log(spew.Sdump(got))

	{
		gotModuleDirs := map[string]string{}
		for k, m := range got.Modules {
			gotModuleDirs[k] = m.Dir
		}
		wantModuleDirs := map[string]string{
			"":                "testdata/config-graph/already-installed",
			"child_a":         "testdata/config-graph/already-installed/.terraform/modules/child_a",
			"child_a.child_c": "testdata/config-graph/already-installed/.terraform/modules/child_a/child_c",
			"child_b":         "testdata/config-graph/already-installed/.terraform/modules/child_b",
			"child_b.child_d": "testdata/config-graph/already-installed/.terraform/modules/child_b.child_d",
		}

		problems := deep.Equal(wantModuleDirs, gotModuleDirs)
		for _, problem := range problems {
			t.Error(problem)
		}
		if len(problems) > 0 {
			return
		}
	}

	gotRoot := got.Modules[""]
	wantRoot := &configload.SnapshotModule{
		Dir: "testdata/config-graph/already-installed",
		Files: map[string][]byte{
			"root.tf": []byte(`
module "child_a" {
  source  = "example.com/foo/bar_a/baz"
  version = ">= 1.0.0"
}

module "child_b" {
  source = "example.com/foo/bar_b/baz"
  version = ">= 1.0.0"
}
`),
		},
	}
	if !reflect.DeepEqual(gotRoot, wantRoot) {
		t.Errorf("wrong root module snapshot\ngot: %swant: %s", spew.Sdump(gotRoot), spew.Sdump(wantRoot))
	}

}

func TestLoadConfigWithSnapshot_invalidSource(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/config-graph/already-installed-now-invalid")

	old, _ := os.Getwd()
	os.Chdir(fixtureDir)
	defer os.Chdir(old)

	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: ".terraform/modules",
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	_, _, diags := testLoadWithSnapshot(".", loader, nil)
	if !diags.HasErrors() {
		t.Error("LoadConfigWithSnapshot succeeded; want errors")
	}
}

func TestSnapshotRoundtrip(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/config-graph/already-installed")
	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	_, snap, diags := testLoadWithSnapshot(fixtureDir, loader, nil)
	assertNoDiagnostics(t, diags)
	if snap == nil {
		t.Fatalf("snapshot is nil; want non-nil")
	}

	snapLoader := configload.NewLoaderFromSnapshot(snap)
	if loader == nil {
		t.Fatalf("loader is nil; want non-nil")
	}
	rootMod, rootDiags := snapLoader.LoadRootModule(snap.Modules[""].Dir)
	assertNoDiagnostics(t, rootDiags)

	config, diags := BuildConfigWithGraph(
		rootMod,
		snapLoader.ModuleWalker(),
		nil,
		configs.MockDataLoaderFunc(snapLoader.LoadExternalMockData),
	)
	assertNoDiagnostics(t, diags)
	if config == nil {
		t.Fatalf("config is nil; want non-nil")
	}
	if config.Module == nil {
		t.Fatalf("config has no root module")
	}
	if got, want := config.Module.SourceDir, "testdata/config-graph/already-installed"; got != want {
		t.Errorf("wrong root module sourcedir %q; want %q", got, want)
	}
	if got, want := len(config.Module.ModuleCalls), 2; got != want {
		t.Errorf("wrong number of module calls in root module %d; want %d", got, want)
	}
	childA := config.Children["child_a"]
	if childA == nil {
		t.Fatalf("child_a config is nil; want non-nil")
	}
	if childA.Module == nil {
		t.Fatalf("child_a config has no module")
	}
	if got, want := childA.Module.SourceDir, "testdata/config-graph/already-installed/.terraform/modules/child_a"; got != want {
		t.Errorf("wrong child_a sourcedir %q; want %q", got, want)
	}
	if got, want := len(childA.Module.ModuleCalls), 1; got != want {
		t.Errorf("wrong number of module calls in child_a %d; want %d", got, want)
	}
}

func TestBuildConfigWithGraph_okay(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/config-graph/already-installed")
	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	cfg, _, diags := testLoadWithSnapshot(fixtureDir, loader, nil)
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
		if !got.RawEquals(cty.StringVal("Hello from child_c")) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, cty.StringVal("Hello from child_c"))
		}
	})
	t.Run("child_b.child_d output", func(t *testing.T) {
		output := cfg.Children["child_b"].Children["child_d"].Module.Outputs["hello"]
		got, diags := output.Expr.Value(nil)
		assertNoDiagnostics(t, diags)
		if !got.RawEquals(cty.StringVal("Hello from child_d")) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, cty.StringVal("Hello from child_d"))
		}
	})
}

func TestBuildConfigWithGraph_loadDiags(t *testing.T) {
	// building a config which didn't load correctly may cause configs to panic
	fixtureDir := filepath.Clean("testdata/config-graph/invalid-names")
	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	rootMod, rootDiags := loader.LoadRootModule(fixtureDir)
	if !rootDiags.HasErrors() {
		t.Fatal("success; want error")
	}

	if rootMod == nil {
		t.Fatal("partial module not returned with diagnostics")
	}
}

func TestBuildConfigWithGraph_loadDiagsFromSubmodules(t *testing.T) {
	// building a config which didn't load correctly may cause configs to panic
	fixtureDir := filepath.Clean("testdata/config-graph/invalid-names-in-submodules")
	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	rootMod, rootDiags := loader.LoadRootModule(fixtureDir)
	if rootDiags.HasErrors() {
		t.Fatalf("unexpected root module load error: %s", rootDiags.Error())
	}

	_, diags := BuildConfigWithGraph(
		rootMod,
		loader.ModuleWalker(),
		nil,
		configs.MockDataLoaderFunc(loader.LoadExternalMockData),
	)
	if !diags.HasErrors() {
		t.Fatalf("loading succeeded; want an error")
	}
	if got, want := diags.Err().Error(), " Invalid provider local name"; !strings.Contains(got, want) {
		t.Errorf("missing expected error\nwant substring: %s\ngot: %s", want, got)
	}
}

func TestBuildConfigWithGraph_childProviderGrandchildCount(t *testing.T) {
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
		fixtureDir := filepath.Clean("testdata/config-graph/child-provider-grandchild-count")
		loader, err := configload.NewLoader(&configload.Config{
			ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
		})
		if err != nil {
			t.Fatalf("unexpected error from NewLoader: %s", err)
		}

		cfg, _, diags := testLoadWithSnapshot(fixtureDir, loader, nil)
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
		fixtureDir := filepath.Clean("testdata/config-graph/child-provider-child-count")
		loader, err := configload.NewLoader(&configload.Config{
			ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
		})
		if err != nil {
			t.Fatalf("unexpected error from NewLoader: %s", err)
		}

		_, _, diags := testLoadWithSnapshot(fixtureDir, loader, nil)
		if !diags.HasErrors() {
			t.Fatalf("loading succeeded; want an error")
		}
		if got, want := diags.Err().Error(), "Module is incompatible with count, for_each, and depends_on"; !strings.Contains(got, want) {
			t.Errorf("missing expected error\nwant substring: %s\ngot: %s", want, got)
		}
	})
}

func assertNoDiagnostics[D hcl.Diagnostics | tfdiags.Diagnostics](t *testing.T, diags D) bool {
	t.Helper()

	if len(diags) != 0 {
		t.Errorf("wrong number of diagnostics %d; want %d", len(diags), 0)
		return true
	}
	return false
}
