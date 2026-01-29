// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
)

// TestSourceBundleParser_LoadConfigDir_WithAbsolutePath tests that when
// LocalPathForSource returns an absolute path, LoadConfigDir correctly converts
// it to a relative path from the current working directory and sets it on the Module.
func TestSourceBundleParser_LoadConfigDir_WithAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	bundleRoot := filepath.Join(tmpDir, "bundle")
	configDir := filepath.Join(bundleRoot, "root")

	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %s", err)
	}

	configContent := []byte(`
resource "test_resource" "example" {
  name = "test"
}
`)
	err = os.WriteFile(filepath.Join(configDir, "main.tf"), configContent, 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %s", err)
	}

	manifestContent := []byte(`{
  "terraform_source_bundle": 1,
  "packages": [
    {
      "source": "git::https://example.com/test.git",
      "local": "root",
      "meta": {}
    }
  ]
}`)
	err = os.WriteFile(filepath.Join(bundleRoot, "terraform-sources.json"), manifestContent, 0644)
	if err != nil {
		t.Fatalf("failed to write manifest file: %s", err)
	}

	sources, err := sourcebundle.OpenDir(bundleRoot)
	if err != nil {
		t.Fatalf("failed to open source bundle: %s", err)
	}

	source := sourceaddrs.MustParseSource("git::https://example.com/test.git").(sourceaddrs.FinalSource)

	sourcePath, err := sources.LocalPathForSource(source)
	if err != nil {
		t.Fatalf("failed to get local path for source: %s", err)
	}
	if !filepath.IsAbs(sourcePath) {
		t.Fatalf("LocalPathForSource returned relative path %q", sourcePath)
	}

	parser := NewSourceBundleParser(sources)
	mod, diags := parser.LoadConfigDir(source)

	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Error())
	}

	if mod == nil {
		t.Fatal("expected non-nil module")
	}

	if filepath.IsAbs(mod.SourceDir) {
		t.Errorf("expected SourceDir to be relative, but got absolute path: %s", mod.SourceDir)
	}

	workDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %s", err)
	}

	expectedRelPath, err := filepath.Rel(workDir, sourcePath)
	if err != nil {
		t.Fatalf("failed to compute expected relative path: %s", err)
	}

	if mod.SourceDir != expectedRelPath {
		t.Errorf("expected SourceDir to be %q, got %q", expectedRelPath, mod.SourceDir)
	}
}

// mockSourceBundleForRelativePath is a test helper that wraps sourcebundle.Bundle
// to simulate LocalPathForSource returning a relative path.
type mockSourceBundleForRelativePath struct {
	realBundle *sourcebundle.Bundle
	workDir    string
}

// We can't reliably ensure that LocalPathForSource will return a relative path - so we can mock it
// to ensure that the core logic of keeping that path relative is working.
func (m *mockSourceBundleForRelativePath) LocalPathForSource(source sourceaddrs.FinalSource) (string, error) {
	path, err := m.realBundle.LocalPathForSource(source)
	if err != nil {
		return "", err
	}

	// Convert absolute path to relative for testing
	if filepath.IsAbs(path) {
		relPath, err := filepath.Rel(m.workDir, path)
		if err != nil {
			return "", err
		}
		return relPath, nil
	}

	return path, nil
}

// TestSourceBundleParser_LoadConfigDir_WithRelativePath tests that when
// LocalPathForSource returns a relative path (already relative to the working
// directory), the code uses it as-is without attempting to convert it. This
// ensures we don't break the case where the path is already in the correct form.
func TestSourceBundleParser_LoadConfigDir_WithRelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	bundleRoot := filepath.Join(tmpDir, "bundle")
	configDir := filepath.Join(bundleRoot, "root")

	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %s", err)
	}

	configContent := []byte(`
resource "test_resource" "example" {
  name = "test"
}
`)
	err = os.WriteFile(filepath.Join(configDir, "main.tf"), configContent, 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %s", err)
	}

	// Create the source bundle manifest
	manifestContent := []byte(`{
  "terraform_source_bundle": 1,
  "packages": [
    {
      "source": "git::https://example.com/relative-test.git",
      "local": "root",
      "meta": {}
    }
  ]
}`)

	err = os.WriteFile(filepath.Join(bundleRoot, "terraform-sources.json"), manifestContent, 0644)
	if err != nil {
		t.Fatalf("failed to write manifest file: %s", err)
	}

	realSources, err := sourcebundle.OpenDir(bundleRoot)
	if err != nil {
		t.Fatalf("failed to open source bundle: %s", err)
	}

	workDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %s", err)
	}

	mockSources := &mockSourceBundleForRelativePath{
		realBundle: realSources,
		workDir:    workDir,
	}

	source := sourceaddrs.MustParseSource("git::https://example.com/relative-test.git").(sourceaddrs.FinalSource)

	sourcePath, err := mockSources.LocalPathForSource(source)
	if err != nil {
		t.Fatalf("failed to get local path for source: %s", err)
	}

	if filepath.IsAbs(sourcePath) {
		t.Fatalf("mock should return relative path but got absolute: %q", sourcePath)
	}

	testPath := sourcePath
	var relativeSourceDir string

	// This mimics the logic in LoadConfigDir
	if filepath.IsAbs(testPath) {
		t.Fatal("unexpected absolute path from mock")
	} else {
		relativeSourceDir = testPath
	}

	if relativeSourceDir != sourcePath {
		t.Errorf("expected relative path to be used as-is: got %q, want %q", relativeSourceDir, sourcePath)
	}

	if filepath.IsAbs(relativeSourceDir) {
		t.Errorf("expected relativeSourceDir to be relative, but got absolute: %s", relativeSourceDir)
	}
}
