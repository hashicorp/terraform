package configload

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs"
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

func TestLoaderLoadConfig_moduleExpand(t *testing.T) {
	// We do not allow providers to be configured in expanding modules
	// In addition, if a provider is present but an empty block, it is allowed,
	// but IFF a provider is passed through the module call
	paths := []string{"provider-configured", "no-provider-passed", "nested-provider", "more-nested-provider"}
	for _, p := range paths {
		fixtureDir := filepath.Clean(fmt.Sprintf("testdata/expand-modules/%s", p))
		loader, err := NewLoader(&Config{
			ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
		})
		if err != nil {
			t.Fatalf("unexpected error from NewLoader at path %s: %s", p, err)
		}

		_, diags := loader.LoadConfig(fixtureDir)
		if !diags.HasErrors() {
			t.Fatalf("success; want error at path %s", p)
		}
		got := diags.Error()
		want := "Module does not support count"
		if !strings.Contains(got, want) {
			t.Fatalf("wrong error at path %s \ngot:\n%s\n\nwant: containing %q", p, got, want)
		}
	}
}

func TestLoaderLoadConfig_moduleExpandDoubleAlias(t *testing.T) {
	// This tests for when a module calls another module, and passes in
	// the correct alias the child is expecting.
	// https://github.com/hashicorp/terraform/issues/27539
	fixtureDir := filepath.Clean("testdata/expand-modules/alias-renamed-twice")
	loader, err := NewLoader(&Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	_, diags := loader.LoadConfig(fixtureDir)
	assertNoDiagnostics(t, diags)
}

func TestLoaderLoadConfig_moduleExpandValid(t *testing.T) {
	// This tests for when valid configs are passing a provider through as a proxy,
	// either with or without an alias present.
	fixtureDir := filepath.Clean("testdata/expand-modules/valid")
	loader, err := NewLoader(&Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	_, diags := loader.LoadConfig(fixtureDir)
	assertNoDiagnostics(t, diags)
}

func TestLoaderLoadConfig_moduleDependsOnProviders(t *testing.T) {
	// We do not allow providers to be configured in module using depends_on.
	fixtureDir := filepath.Clean("testdata/module-depends-on")
	loader, err := NewLoader(&Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform/modules"),
	})
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	_, diags := loader.LoadConfig(fixtureDir)
	if !diags.HasErrors() {
		t.Fatal("success; want error")
	}
	got := diags.Error()
	want := "Module does not support depends_on"
	if !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot:\n%s\n\nwant: containing %q", got, want)
	}
}
