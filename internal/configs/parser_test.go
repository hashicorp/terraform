// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/spf13/afero"
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
	cfg, moreDiags := BuildConfig(mod, nil)
	return cfg, append(diags, moreDiags...)
}

// testModuleFromDir reads configuration from the given directory path as
// a module and returns it. This is a helper for use in unit tests.
func testModuleFromDir(path string) (*Module, hcl.Diagnostics) {
	parser := NewParser(nil)
	return parser.LoadConfigDir(path)
}

// testModuleFromDir reads configuration from the given directory path as a
// module and returns its configuration. This is a helper for use in unit tests.
func testModuleConfigFromDir(path string) (*Config, hcl.Diagnostics) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDir(path)
	cfg, moreDiags := BuildConfig(mod, nil)
	return cfg, append(diags, moreDiags...)
}

// testNestedModuleConfigFromDirWithTests matches testNestedModuleConfigFromDir
// except it also loads any test files within the directory.
func testNestedModuleConfigFromDirWithTests(t *testing.T, path string) (*Config, hcl.Diagnostics) {
	t.Helper()

	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDirWithTests(path, "tests")
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
	versionI := 0
	return BuildConfig(mod, ModuleWalkerFunc(
		func(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics) {
			// For the sake of this test we're going to just treat our
			// SourceAddr as a path relative to the calling module.
			// A "real" implementation of ModuleWalker should accept the
			// various different source address syntaxes Terraform supports.

			// Build a full path by walking up the module tree, prepending each
			// source address path until we hit the root
			paths := []string{req.SourceAddr.String()}
			for config := req.Parent; config != nil && config.Parent != nil; config = config.Parent {
				paths = append([]string{config.SourceAddr.String()}, paths...)
			}
			paths = append([]string{path}, paths...)
			sourcePath := filepath.Join(paths...)

			mod, diags := parser.LoadConfigDir(sourcePath)
			version, _ := version.NewVersion(fmt.Sprintf("1.0.%d", versionI))
			versionI++
			return mod, version, diags
		},
	))
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
