// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package staticmodule

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/configs"
)

func TestBuildConfig(t *testing.T) {
	parser := configs.NewParser(nil)
	dir := "testdata/config-build"
	mod, diags := parser.LoadConfigDir(dir)
	assertNoDiagnostics(t, diags)

	versionI := 0
	cfg, diags := BuildConfig(mod, configs.ModuleWalkerFunc(func(req *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {
		mod, diags := parser.LoadConfigDir(filepath.Join(dir, req.SourceAddr.String()))
		ver, _ := version.NewVersion(fmt.Sprintf("1.0.%d", versionI))
		versionI++
		return mod, ver, diags
	}))
	assertNoDiagnostics(t, diags)

	var got []string
	cfg.DeepEach(func(c *configs.Config) {
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
		t.Fatalf("wrong configuration tree\ngot:  %#v\nwant: %#v", got, want)
	}

	childA := cfg.Children["child_a"].Children["child_c"].Module
	childB := cfg.Children["child_b"].Children["child_c"].Module
	if _, exists := childA.Outputs["hello"]; !exists {
		t.Fatal("missing output hello in child_a.child_c")
	}
	if _, exists := childB.Outputs["hello"]; !exists {
		t.Fatal("missing output hello in child_b.child_c")
	}
	if childA == childB {
		t.Fatal("child_a.child_c and child_b.child_c should be distinct module instances")
	}
}

func TestBuildConfigPreservesWalkerDiagnostics(t *testing.T) {
	parser := configs.NewParser(nil)
	dir := "testdata/nested-errors"
	mod, diags := parser.LoadConfigDir(dir)
	assertNoDiagnostics(t, diags)

	cfg, diags := BuildConfig(mod, fixtureWalker(parser, dir))
	if !diagnosticSummaryExists(diags, "Unsupported block type") {
		t.Fatalf("missing child module diagnostic: %s", diags.Error())
	}
	if cfg.Children["child_a"].Children["child_c"] == nil {
		t.Fatal("configuration tree should include the child module that returned diagnostics")
	}
}

func TestBuildConfigChildModuleWarnings(t *testing.T) {
	tests := map[string]string{
		"backend": "Backend configuration ignored",
		"cloud":   "Cloud configuration ignored",
	}
	for name, wantSummary := range tests {
		t.Run(name, func(t *testing.T) {
			parser := configs.NewParser(nil)
			dir := filepath.Join("testdata", "nested-"+name+"-warning")
			mod, diags := parser.LoadConfigDir(dir)
			assertNoDiagnostics(t, diags)

			cfg, diags := BuildConfig(mod, fixtureWalker(parser, dir))
			if !diagnosticSummaryExists(diags, wantSummary) {
				t.Fatalf("missing %q diagnostic: %s", wantSummary, diags.Error())
			}
			if cfg.Children["child"] == nil {
				t.Fatal("configuration tree should include child module")
			}
		})
	}
}

func TestBuildConfigRejectsListInChildModule(t *testing.T) {
	parser := configs.NewParser(nil)
	dir := "testdata/list-in-child-module"
	mod, diags := parser.LoadConfigDir(dir, configs.MatchQueryFiles())
	assertNoDiagnostics(t, diags)

	_, diags = BuildConfig(mod, configs.ModuleWalkerFunc(func(req *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {
		mod, diags := parser.LoadConfigDir(filepath.Join(dir, req.SourceAddr.String()), configs.MatchQueryFiles())
		return mod, nil, diags
	}))
	if !diagnosticSummaryExists(diags, "Invalid list configuration") {
		t.Fatalf("missing child list diagnostic: %s", diags.Error())
	}
}

func fixtureWalker(parser *configs.Parser, dir string) configs.ModuleWalker {
	return configs.ModuleWalkerFunc(func(req *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {
		mod, diags := parser.LoadConfigDir(filepath.Join(dir, req.SourceAddr.String()))
		ver, _ := version.NewVersion("1.0.0")
		return mod, ver, diags
	})
}

func assertNoDiagnostics(t *testing.T, diags hcl.Diagnostics) {
	t.Helper()
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %s", diags.Error())
	}
}

func diagnosticSummaryExists(diags hcl.Diagnostics, summary string) bool {
	for _, diag := range diags {
		if diag.Summary == summary {
			return true
		}
	}
	return false
}
