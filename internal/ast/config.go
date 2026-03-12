// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ast

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/schemarepo"
)

// BlockResult pairs a found Block with the File and Module it belongs to.
type BlockResult struct {
	Block  *Block
	File   *File
	Module *Module
}

// Module holds all writable HCL files for a single module directory.
type Module struct {
	files    []*File
	path     string
	editable bool
	schemas  *schemarepo.Schemas
}

// NewModule creates a Module from pre-parsed files.
func NewModule(files []*File, path string, editable bool, schemas *schemarepo.Schemas) *Module {
	return &Module{
		files:    files,
		path:     path,
		editable: editable,
		schemas:  schemas,
	}
}

// Path returns the filesystem directory this module was loaded from.
func (m *Module) Path() string {
	return m.path
}

// Editable returns whether this module can be mutated.
func (m *Module) Editable() bool {
	return m.editable
}

// FindBlocks searches all files in this module for blocks matching
// blockType and firstLabel. If firstLabel is empty, matches blocks
// with no labels (e.g., terraform {}).
func (m *Module) FindBlocks(blockType, firstLabel string) []*BlockResult {
	var results []*BlockResult
	for _, f := range m.files {
		for _, b := range f.FindBlocks(blockType, firstLabel) {
			results = append(results, &BlockResult{
				Block:  b,
				File:   f,
				Module: m,
			})
		}
	}
	return results
}

// Bytes returns filename->content for all files in this module.
// Panics if the module is not editable.
func (m *Module) Bytes() map[string][]byte {
	if !m.editable {
		panic("ast: Bytes() called on non-editable module at path " + m.path)
	}
	result := make(map[string][]byte, len(m.files))
	for _, f := range m.files {
		result[f.filename] = f.Bytes()
	}
	return result
}

// ModuleNode is a node in the module tree. Each node holds the
// parsed writable files for one module directory.
type ModuleNode struct {
	Module   *Module
	Children map[string]*ModuleNode
	Parent   *ModuleNode
}

// Child returns the child module node by name, or nil if not found.
func (n *ModuleNode) Child(name string) *ModuleNode {
	return n.Children[name]
}

// Walk visits this node and all descendants depth-first, calling
// fn for each node.
func (n *ModuleNode) Walk(fn func(*ModuleNode)) {
	fn(n)
	for _, child := range n.Children {
		child.Walk(fn)
	}
}

// Config wraps a full Terraform configuration tree for migration.
type Config struct {
	Root *ModuleNode
}

// FindBlocks searches all editable modules in the tree for blocks
// matching blockType and firstLabel.
func (c *Config) FindBlocks(blockType, firstLabel string) []*BlockResult {
	var results []*BlockResult
	c.Root.Walk(func(n *ModuleNode) {
		if !n.Module.Editable() {
			return
		}
		results = append(results, n.Module.FindBlocks(blockType, firstLabel)...)
	})
	return results
}

// LoadConfig walks a configs.Config tree, re-parses each module's .tf files
// with hclwrite, and builds a Config tree for migration operations.
func LoadConfig(cfg *configs.Config, schemas *schemarepo.Schemas) (*Config, error) {
	root, err := buildModuleNode(cfg, nil, schemas)
	if err != nil {
		return nil, err
	}
	return &Config{Root: root}, nil
}

// buildModuleNode recursively builds a ModuleNode from a configs.Config.
func buildModuleNode(cfg *configs.Config, parent *ModuleNode, schemas *schemarepo.Schemas) (*ModuleNode, error) {
	editable := isEditable(cfg)

	files, err := parseModuleFiles(cfg.Module.SourceDir)
	if err != nil {
		return nil, fmt.Errorf("loading module %s: %w", cfg.Module.SourceDir, err)
	}

	mod := NewModule(files, cfg.Module.SourceDir, editable, schemas)

	node := &ModuleNode{
		Module:   mod,
		Children: make(map[string]*ModuleNode),
		Parent:   parent,
	}

	for name, childCfg := range cfg.Children {
		childNode, err := buildModuleNode(childCfg, node, schemas)
		if err != nil {
			return nil, err
		}
		node.Children[name] = childNode
	}

	return node, nil
}

// isEditable determines if a module is editable based on its source type.
// The root module (no SourceAddr) and local modules are editable.
func isEditable(cfg *configs.Config) bool {
	if cfg.SourceAddr == nil {
		return true // root module
	}
	_, isLocal := cfg.SourceAddr.(addrs.ModuleSourceLocal)
	return isLocal
}

// parseModuleFiles discovers and parses all .tf files in a directory,
// skipping editor swap files and other ignored files.
func parseModuleFiles(dir string) ([]*File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var files []*File
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".tf") || configs.IsIgnoredFile(name) {
			continue
		}
		path := filepath.Join(dir, name)
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		f, err := ParseFile(src, path, nil)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

// RenameReferencePrefix renames variable reference prefixes across
// all files in this module.
func (m *Module) RenameReferencePrefix(old, new hcl.Traversal) {
	for _, f := range m.files {
		f.RenameReferencePrefix(old, new)
	}
}

// RenameBlockType renames all top-level blocks matching oldType to
// newType across all files in this module.
func (m *Module) RenameBlockType(oldType, newType string) {
	for _, f := range m.files {
		f.RenameBlockType(oldType, newType)
	}
}
