// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ast

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	svchost "github.com/hashicorp/terraform-svchost"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

func TestNewModule(t *testing.T) {
	fileA, err := ParseFile([]byte(`resource "aws_instance" "a" {
  ami = "abc"
}
`), "a.tf", nil)
	if err != nil {
		t.Fatal(err)
	}
	fileB, err := ParseFile([]byte(`resource "aws_instance" "b" {
  ami = "def"
}
`), "b.tf", nil)
	if err != nil {
		t.Fatal(err)
	}

	mod := NewModule([]*File{fileA, fileB}, "root", true, nil)

	if mod.Path() != "root" {
		t.Errorf("Path() = %q, want %q", mod.Path(), "root")
	}
	if !mod.Editable() {
		t.Error("expected module to be editable")
	}
}

func TestModule_FindBlocks(t *testing.T) {
	fileA, _ := ParseFile([]byte(`resource "aws_instance" "a" {
  ami = "abc"
}

resource "aws_s3_bucket" "data" {
  bucket = "my-bucket"
}
`), "main.tf", nil)
	fileB, _ := ParseFile([]byte(`resource "aws_instance" "b" {
  ami = "def"
}

terraform {
  required_version = ">= 1.0"
}
`), "providers.tf", nil)

	mod := NewModule([]*File{fileA, fileB}, "", true, nil)

	tests := map[string]struct {
		blockType  string
		firstLabel string
		wantCount  int
	}{
		"finds across files": {
			blockType:  "resource",
			firstLabel: "aws_instance",
			wantCount:  2,
		},
		"finds in single file": {
			blockType:  "resource",
			firstLabel: "aws_s3_bucket",
			wantCount:  1,
		},
		"no matches": {
			blockType:  "resource",
			firstLabel: "aws_lambda_function",
			wantCount:  0,
		},
		"finds label-less blocks": {
			blockType:  "terraform",
			firstLabel: "",
			wantCount:  1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			results := mod.FindBlocks(tc.blockType, tc.firstLabel)
			if len(results) != tc.wantCount {
				t.Fatalf("got %d results, want %d", len(results), tc.wantCount)
			}
			for _, r := range results {
				if r.Module != mod {
					t.Error("BlockResult.Module does not point to the source module")
				}
				if r.File == nil {
					t.Error("BlockResult.File is nil")
				}
				if r.Block == nil {
					t.Error("BlockResult.Block is nil")
				}
			}
		})
	}
}

func TestModule_Bytes(t *testing.T) {
	fileA, _ := ParseFile([]byte(`resource "a" "b" {
}
`), "a.tf", nil)
	fileB, _ := ParseFile([]byte(`resource "c" "d" {
}
`), "b.tf", nil)

	mod := NewModule([]*File{fileA, fileB}, "", true, nil)
	bytesMap := mod.Bytes()

	if len(bytesMap) != 2 {
		t.Fatalf("got %d files, want 2", len(bytesMap))
	}
	if _, ok := bytesMap["a.tf"]; !ok {
		t.Error("missing a.tf")
	}
	if _, ok := bytesMap["b.tf"]; !ok {
		t.Error("missing b.tf")
	}
}

func TestModule_Bytes_panics_on_non_editable(t *testing.T) {
	mod := NewModule(nil, "external", false, nil)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got none")
		}
	}()

	mod.Bytes()
}

func TestModule_RenameReferencePrefix(t *testing.T) {
	fileA, _ := ParseFile([]byte(`resource "aws_instance" "a" {
  bucket = aws_s3_bucket.main.id
}
`), "main.tf", nil)
	fileB, _ := ParseFile([]byte(`output "bucket_id" {
  value = aws_s3_bucket.main.id
}
`), "outputs.tf", nil)

	mod := NewModule([]*File{fileA, fileB}, "", true, nil)
	mod.RenameReferencePrefix(
		makeTraversal("aws_s3_bucket"),
		makeTraversal("aws_bucket"),
	)

	bytesMap := mod.Bytes()
	mainContent := string(bytesMap["main.tf"])
	if !strings.Contains(mainContent, "aws_bucket.main.id") {
		t.Errorf("main.tf not renamed:\n%s", mainContent)
	}
	outputContent := string(bytesMap["outputs.tf"])
	if !strings.Contains(outputContent, "aws_bucket.main.id") {
		t.Errorf("outputs.tf not renamed:\n%s", outputContent)
	}
}

func TestModuleNode_Walk(t *testing.T) {
	rootMod := NewModule(nil, "", true, nil)
	childAMod := NewModule(nil, "child_a", true, nil)
	childBMod := NewModule(nil, "child_b", false, nil)
	grandchildMod := NewModule(nil, "child_a.nested", true, nil)

	grandchild := &ModuleNode{Module: grandchildMod, Children: map[string]*ModuleNode{}}
	childA := &ModuleNode{
		Module:   childAMod,
		Children: map[string]*ModuleNode{"nested": grandchild},
	}
	grandchild.Parent = childA

	childB := &ModuleNode{Module: childBMod, Children: map[string]*ModuleNode{}}
	root := &ModuleNode{
		Module: rootMod,
		Children: map[string]*ModuleNode{
			"child_a": childA,
			"child_b": childB,
		},
	}
	childA.Parent = root
	childB.Parent = root

	// Test Child
	if got := root.Child("child_a"); got != childA {
		t.Error("Child(child_a) returned wrong node")
	}
	if got := root.Child("nonexistent"); got != nil {
		t.Error("Child(nonexistent) should return nil")
	}

	// Test Walk visits all nodes
	var visited []string
	root.Walk(func(n *ModuleNode) {
		visited = append(visited, n.Module.Path())
	})
	if len(visited) != 4 {
		t.Fatalf("Walk visited %d nodes, want 4: %v", len(visited), visited)
	}

	// Test Parent
	if grandchild.Parent != childA {
		t.Error("grandchild.Parent should be childA")
	}
	if childA.Parent != root {
		t.Error("childA.Parent should be root")
	}
	if root.Parent != nil {
		t.Error("root.Parent should be nil")
	}
}

func TestConfig_FindBlocks(t *testing.T) {
	rootFileA, _ := ParseFile([]byte(`resource "aws_instance" "root_a" {
  ami = "abc"
}
`), "main.tf", nil)
	rootFileB, _ := ParseFile([]byte(`terraform {
  required_version = ">= 1.0"
}
`), "versions.tf", nil)
	childFile, _ := ParseFile([]byte(`resource "aws_instance" "child_a" {
  ami = "def"
}
`), "main.tf", nil)
	externalFile, _ := ParseFile([]byte(`resource "aws_instance" "ext" {
  ami = "ghi"
}
`), "main.tf", nil)

	rootMod := NewModule([]*File{rootFileA, rootFileB}, "", true, nil)
	childMod := NewModule([]*File{childFile}, "child", true, nil)
	externalMod := NewModule([]*File{externalFile}, "external", false, nil)

	cfg := &Config{
		Root: &ModuleNode{
			Module: rootMod,
			Children: map[string]*ModuleNode{
				"child": {
					Module:   childMod,
					Children: map[string]*ModuleNode{},
				},
				"external": {
					Module:   externalMod,
					Children: map[string]*ModuleNode{},
				},
			},
		},
	}

	tests := map[string]struct {
		blockType  string
		firstLabel string
		wantCount  int
	}{
		"finds across editable modules only": {
			blockType:  "resource",
			firstLabel: "aws_instance",
			wantCount:  2, // root_a + child_a, NOT ext
		},
		"no matches": {
			blockType:  "resource",
			firstLabel: "aws_lambda_function",
			wantCount:  0,
		},
		"finds label-less blocks": {
			blockType:  "terraform",
			firstLabel: "",
			wantCount:  1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			results := cfg.FindBlocks(tc.blockType, tc.firstLabel)
			if len(results) != tc.wantCount {
				t.Fatalf("got %d results, want %d", len(results), tc.wantCount)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory structure:
	// root/
	//   main.tf
	//   child/        (local module)
	//     main.tf
	rootDir := t.TempDir()
	childDir := filepath.Join(rootDir, "child")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}

	rootContent := []byte(`resource "aws_instance" "root" {
  ami = "abc"
}
`)
	childContent := []byte(`resource "aws_instance" "child" {
  ami = "def"
}
`)
	if err := os.WriteFile(filepath.Join(rootDir, "main.tf"), rootContent, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childDir, "main.tf"), childContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Build a minimal configs.Config tree
	rootCfg := &configs.Config{
		Module: &configs.Module{
			SourceDir: rootDir,
		},
		Children: map[string]*configs.Config{
			"child": {
				Module: &configs.Module{
					SourceDir: childDir,
				},
				SourceAddr: addrs.ModuleSourceLocal("./child"),
				Children:   map[string]*configs.Config{},
			},
		},
	}

	cfg, err := LoadConfig(rootCfg, nil)
	if err != nil {
		t.Fatalf("LoadConfig() error: %s", err)
	}

	// Root module should be editable
	if !cfg.Root.Module.Editable() {
		t.Error("root module should be editable")
	}

	// Should find resources across both modules
	results := cfg.FindBlocks("resource", "aws_instance")
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// Child should be accessible
	childNode := cfg.Root.Child("child")
	if childNode == nil {
		t.Fatal("child node not found")
	}
	if !childNode.Module.Editable() {
		t.Error("local child module should be editable")
	}
	if childNode.Parent != cfg.Root {
		t.Error("child.Parent should point to root")
	}
}

func TestLoadConfig_external_module_not_editable(t *testing.T) {
	rootDir := t.TempDir()
	externalDir := filepath.Join(rootDir, ".terraform", "modules", "vpc")
	if err := os.MkdirAll(externalDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(rootDir, "main.tf"), []byte(`module "vpc" {
  source  = "hashicorp/vpc/aws"
  version = "3.0.0"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(externalDir, "main.tf"), []byte(`resource "aws_vpc" "this" {
  cidr_block = "10.0.0.0/16"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCfg := &configs.Config{
		Module: &configs.Module{
			SourceDir: rootDir,
		},
		Children: map[string]*configs.Config{
			"vpc": {
				Module: &configs.Module{
					SourceDir: externalDir,
				},
				SourceAddr: addrs.ModuleSourceRegistry{
					Package: addrs.ModuleRegistryPackage{
						Host:         svchost.Hostname("registry.terraform.io"),
						Namespace:    "hashicorp",
						Name:         "vpc",
						TargetSystem: "aws",
					},
				},
				Children: map[string]*configs.Config{},
			},
		},
	}

	cfg, err := LoadConfig(rootCfg, nil)
	if err != nil {
		t.Fatalf("LoadConfig() error: %s", err)
	}

	// External module should exist but not be editable
	vpcNode := cfg.Root.Child("vpc")
	if vpcNode == nil {
		t.Fatal("vpc child node not found")
	}
	if vpcNode.Module.Editable() {
		t.Error("registry module should not be editable")
	}

	// FindBlocks should only return root module blocks
	results := cfg.FindBlocks("resource", "aws_vpc")
	if len(results) != 0 {
		t.Errorf("got %d results from non-editable module, want 0", len(results))
	}
}

func TestModule_RenameBlockType(t *testing.T) {
	fileA, _ := ParseFile([]byte(`resource "aws_instance" "a" {
  ami = "abc"
}
`), "main.tf", nil)
	fileB, _ := ParseFile([]byte(`resource "aws_instance" "b" {
  ami = "def"
}

data "aws_ami" "latest" {
  most_recent = true
}
`), "other.tf", nil)

	mod := NewModule([]*File{fileA, fileB}, "", true, nil)
	mod.RenameBlockType("resource", "moved")

	bytesMap := mod.Bytes()
	for filename, content := range bytesMap {
		s := string(content)
		if strings.Contains(s, "\nresource ") || strings.HasPrefix(s, "resource ") {
			t.Errorf("%s still contains 'resource':\n%s", filename, s)
		}
	}
	if !strings.Contains(string(bytesMap["other.tf"]), "data") {
		t.Error("data block was incorrectly renamed")
	}
}
