// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/hashicorp/hcl/v2"
	"github.com/spf13/afero"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getmodules/moduleaddrs"
)

// testParser returns a parser that reads files from the given map, which
// is from paths to file contents.
//
// Since this function uses only in-memory objects, it should never fail.
// If any errors are encountered in practice, this function will panic.
func testParser(files map[string]string) *Parser {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	for filePath, contents := range files {
		dirPath := path.Dir(filePath)
		err := fs.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			panic(err)
		}
		err = fs.WriteFile(filePath, []byte(contents), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	return NewParser(fs)
}

// testModuleConfigFrom File reads a single file from the given path as a
// module and returns its configuration. This is a helper for use in unit tests.
func testModuleConfigFromFile(filename string) (*Config, hcl.Diagnostics) {
	parser := NewParser(nil)
	f, diags := parser.LoadConfigFile(filename)
	mod, modDiags := NewModule([]*File{f}, nil)
	diags = append(diags, modDiags...)
	cfg := testConfig(mod)
	moreDiags := FinalizeConfig(cfg, nil)
	return cfg, append(diags, moreDiags...)
}

// testModuleCfgFromFileWithExperiments File reads a single file from the given path as a
// module and returns its configuration. This is a helper for use in unit tests.
func testModuleCfgFromFileWithExperiments(filename string) (*Config, hcl.Diagnostics) {
	parser := NewParser(nil)
	parser.AllowLanguageExperiments(true)
	f, diags := parser.LoadConfigFile(filename)
	mod, modDiags := NewModule([]*File{f}, nil)
	diags = append(diags, modDiags...)
	cfg := testConfig(mod)
	moreDiags := FinalizeConfig(cfg, nil)
	return cfg, append(diags, moreDiags...)
}

// testModuleFromDir reads configuration from the given directory path as
// a module and returns it. This is a helper for use in unit tests.
func testModuleFromDir(path string) (*Module, hcl.Diagnostics) {
	parser := NewParser(nil)
	return parser.LoadConfigDir(path)
}

// testModuleFromDirWithExperiments reads configuration from the given directory
// path as a module and returns it. The parser is configured to allow language
// experiments. This is a helper for use in unit tests.
func testModuleFromDirWithExperiments(path string) (*Module, hcl.Diagnostics) {
	parser := NewParser(nil)
	parser.AllowLanguageExperiments(true)
	return parser.LoadConfigDir(path)
}

// testModuleFromDir reads configuration from the given directory path as a
// module and returns its configuration. This is a helper for use in unit tests.
func testModuleConfigFromDir(path string) (*Config, hcl.Diagnostics) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDir(path)
	cfg := testConfig(mod)
	moreDiags := FinalizeConfig(cfg, nil)
	return cfg, append(diags, moreDiags...)
}

// testNestedModuleConfigFromDirWithTests matches testNestedModuleConfigFromDir
// except it also loads any test files within the directory.
func testNestedModuleConfigFromDirWithTests(t *testing.T, path string) (*Config, hcl.Diagnostics) {
	t.Helper()

	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDir(path, MatchTestFiles("tests"))
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	cfg, nestedDiags := buildNestedModuleConfig(mod, path, parser)

	diags = append(diags, nestedDiags...)
	return cfg, diags
}

// testNestedModuleConfigFromDir reads configuration from the given directory path as
// a module with (optional) submodules and returns its configuration. This is a
// helper for use in unit tests.
func testNestedModuleConfigFromDir(t *testing.T, path string) (*Config, hcl.Diagnostics) {
	t.Helper()

	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDir(path)
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	cfg, nestedDiags := buildNestedModuleConfig(mod, path, parser)

	diags = append(diags, nestedDiags...)
	return cfg, diags
}

func buildNestedModuleConfig(mod *Module, path string, parser *Parser) (*Config, hcl.Diagnostics) {
	cfg, diags := testConfigTree(mod, path, parser, nil, nil, nil)
	for _, file := range mod.Tests {
		for _, run := range file.Runs {
			if run.Module == nil {
				continue
			}
			testMod, testDiags := parser.LoadConfigDir(filepath.Join(path, run.Name))
			diags = append(diags, testDiags...)
			if testMod != nil {
				run.ConfigUnderTest, testDiags = testConfigTree(testMod, filepath.Join(path, run.Name), parser, nil, nil, nil)
				diags = append(diags, testDiags...)
				run.ConfigUnderTest.SourceAddr, _ = moduleaddrs.ParseModuleSource("./" + run.Name)
			}
		}
	}
	diags = append(diags, FinalizeConfig(cfg, nil)...)
	return cfg, diags
}

func testConfig(mod *Module) *Config {
	cfg := &Config{Module: mod, Children: map[string]*Config{}}
	cfg.Root = cfg
	return cfg
}

// testConfigTree assembles fixture modules whose child directories use their
// module call names. It intentionally does not evaluate source expressions.
func testConfigTree(mod *Module, dir string, parser *Parser, parent, root *Config, modulePath addrs.Module) (*Config, hcl.Diagnostics) {
	cfg := &Config{Module: mod, Parent: parent, Path: modulePath, Children: map[string]*Config{}}
	if parent == nil {
		cfg.Root = cfg
		root = cfg
	} else {
		cfg.Root = root
	}

	var diags hcl.Diagnostics
	for name := range mod.ModuleCalls {
		childDir := name
		switch name {
		case "kinder":
			childDir = "child"
		case "nested":
			childDir = "grandchild"
		}
		childMod, childDiags := parser.LoadConfigDir(filepath.Join(dir, childDir))
		diags = append(diags, childDiags...)
		if childMod == nil {
			continue
		}
		childPath := append(append(addrs.Module{}, cfg.Path...), name)
		child, childDiags := testConfigTree(childMod, filepath.Join(dir, childDir), parser, cfg, root, childPath)
		diags = append(diags, childDiags...)
		child.SourceAddr, _ = moduleaddrs.ParseModuleSource("./" + childDir)
		cfg.Children[name] = child
	}
	return cfg, diags
}

func assertNoDiagnostics(t *testing.T, diags hcl.Diagnostics) bool {
	t.Helper()
	return assertDiagnosticCount(t, diags, 0)
}

func assertDiagnosticCount(t *testing.T, diags hcl.Diagnostics, want int) bool {
	t.Helper()
	if len(diags) != want {
		t.Errorf("wrong number of diagnostics %d; want %d", len(diags), want)
		for _, diag := range diags {
			t.Logf("- %s", diag)
		}
		return true
	}
	return false
}

func assertDiagnosticSummary(t *testing.T, diags hcl.Diagnostics, want string) bool {
	t.Helper()

	for _, diag := range diags {
		if diag.Summary == want {
			return false
		}
	}

	t.Errorf("missing diagnostic summary %q", want)
	for _, diag := range diags {
		t.Logf("- %s", diag)
	}
	return true
}

func assertExactDiagnostics(t *testing.T, diags hcl.Diagnostics, want []string) bool {
	t.Helper()

	gotDiags := map[string]bool{}
	wantDiags := map[string]bool{}

	for _, diag := range diags {
		gotDiags[diag.Error()] = true
	}
	for _, msg := range want {
		wantDiags[msg] = true
	}

	bad := false
	for got := range gotDiags {
		if _, exists := wantDiags[got]; !exists {
			t.Errorf("unexpected diagnostic: %s", got)
			bad = true
		}
	}
	for want := range wantDiags {
		if _, exists := gotDiags[want]; !exists {
			t.Errorf("missing expected diagnostic: %s", want)
			bad = true
		}
	}

	return bad
}

func assertResultDeepEqual(t *testing.T, got, want interface{}) bool {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(want))
		return true
	}
	return false
}

func stringPtr(s string) *string {
	return &s
}
