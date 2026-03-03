// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-test/deep"
	"github.com/hashicorp/hcl/v2"
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

func assertNoDiagnostics[D hcl.Diagnostics | tfdiags.Diagnostics](t *testing.T, diags D) bool {
	t.Helper()

	if len(diags) != 0 {
		t.Errorf("wrong number of diagnostics %d; want %d", len(diags), 0)
		return true
	}
	return false
}
