package configs

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl2/hcl"
)

func TestBuildConfig(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDir("test-fixtures/config-build")
	assertNoDiagnostics(t, diags)
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	versionI := 0
	cfg, diags := BuildConfig(mod, ModuleWalkerFunc(
		func(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics) {
			// For the sake of this test we're going to just treat our
			// SourceAddr as a path relative to our fixture directory.
			// A "real" implementation of ModuleWalker should accept the
			// various different source address syntaxes Terraform supports.
			sourcePath := filepath.Join("test-fixtures/config-build", req.SourceAddr)

			mod, diags := parser.LoadConfigDir(sourcePath)
			version, _ := version.NewVersion(fmt.Sprintf("1.0.%d", versionI))
			versionI++
			return mod, version, diags
		},
	))
	assertNoDiagnostics(t, diags)
	if cfg == nil {
		t.Fatal("got nil config; want non-nil")
	}

	var got []string
	cfg.DeepEach(func(c *Config) {
		got = append(got, fmt.Sprintf("%s %s", strings.Join(c.Path, "."), c.Version))
	})
	sort.Strings(got)
	want := []string{
		" <nil>",
		"child_a 1.0.0",
		"child_a.child_c 1.0.1",
		"child_b 1.0.2",
		"child_b.child_c 1.0.3",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("wrong result\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(want))
	}

	if _, exists := cfg.Children["child_a"].Children["child_c"].Module.Outputs["hello"]; !exists {
		t.Fatalf("missing output 'hello' in child_a.child_c")
	}
	if _, exists := cfg.Children["child_b"].Children["child_c"].Module.Outputs["hello"]; !exists {
		t.Fatalf("missing output 'hello' in child_b.child_c")
	}
	if cfg.Children["child_a"].Children["child_c"].Module == cfg.Children["child_b"].Children["child_c"].Module {
		t.Fatalf("child_a.child_c is same object as child_b.child_c; should not be")
	}
}
