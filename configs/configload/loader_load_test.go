package configload

import (
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
	fixtureDir := filepath.Clean("test-fixtures/already-installed")
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
