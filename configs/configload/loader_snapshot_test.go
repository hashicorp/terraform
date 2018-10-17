package configload

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-test/deep"
)

func TestLoadConfigWithSnapshot(t *testing.T) {
	fixtureDir := filepath.Clean("test-fixtures/already-installed")
	loader, err := NewLoader(&Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	_, got, diags := loader.LoadConfigWithSnapshot(fixtureDir)
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
			"":                "test-fixtures/already-installed",
			"child_a":         "test-fixtures/already-installed/.terraform/modules/child_a",
			"child_a.child_c": "test-fixtures/already-installed/.terraform/modules/child_a/child_c",
			"child_b":         "test-fixtures/already-installed/.terraform/modules/child_b",
			"child_b.child_d": "test-fixtures/already-installed/.terraform/modules/child_b.child_d",
		}

		problems := deep.Equal(wantModuleDirs, gotModuleDirs)
		for _, problem := range problems {
			t.Errorf(problem)
		}
		if len(problems) > 0 {
			return
		}
	}

	gotRoot := got.Modules[""]
	wantRoot := &SnapshotModule{
		Dir: "test-fixtures/already-installed",
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

func TestSnapshotRoundtrip(t *testing.T) {
	fixtureDir := filepath.Clean("test-fixtures/already-installed")
	loader, err := NewLoader(&Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	_, snap, diags := loader.LoadConfigWithSnapshot(fixtureDir)
	assertNoDiagnostics(t, diags)
	if snap == nil {
		t.Fatalf("snapshot is nil; want non-nil")
	}

	snapLoader := NewLoaderFromSnapshot(snap)
	if loader == nil {
		t.Fatalf("loader is nil; want non-nil")
	}

	config, diags := snapLoader.LoadConfig(fixtureDir)
	assertNoDiagnostics(t, diags)
	if config == nil {
		t.Fatalf("config is nil; want non-nil")
	}
	if config.Module == nil {
		t.Fatalf("config has no root module")
	}
	if got, want := config.Module.SourceDir, "test-fixtures/already-installed"; got != want {
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
	if got, want := childA.Module.SourceDir, "test-fixtures/already-installed/.terraform/modules/child_a"; got != want {
		t.Errorf("wrong child_a sourcedir %q; want %q", got, want)
	}
	if got, want := len(childA.Module.ModuleCalls), 1; got != want {
		t.Errorf("wrong number of module calls in child_a %d; want %d", got, want)
	}
}
