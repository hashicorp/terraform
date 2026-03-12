// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ast

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
)

// File wraps an hclwrite.File to provide migration-oriented operations
// that preserve comments and formatting.
type File struct {
	file     *hclwrite.File
	filename string
	schemas  *providers.ProviderSchema
}

// Block wraps an hclwrite.Block to provide migration-oriented accessors
// and nested block navigation.
type Block struct {
	block *hclwrite.Block
}

// ParseFile parses HCL source code into a File that supports
// comment-preserving mutations and roundtrip writing.
// The schemas parameter can be nil.
func ParseFile(src []byte, filename string, schemas *providers.ProviderSchema) (*File, error) {
	f, diags := hclwrite.ParseConfig(src, filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing %s: %s", filename, diags.Error())
	}
	return &File{
		file:     f,
		filename: filename,
		schemas:  schemas,
	}, nil
}

// Bytes returns the current content of the file, reflecting any
// mutations that have been applied.
func (f *File) Bytes() []byte {
	return f.file.Bytes()
}

// FindBlocks returns all top-level blocks matching blockType whose first
// label equals firstLabel. For example, FindBlocks("resource", "aws_s3_bucket")
// returns all resource "aws_s3_bucket" blocks regardless of their name label.
func (f *File) FindBlocks(blockType, firstLabel string) []*Block {
	var result []*Block
	for _, block := range f.file.Body().Blocks() {
		if block.Type() != blockType {
			continue
		}
		labels := block.Labels()
		if firstLabel == "" {
			if len(labels) == 0 {
				result = append(result, &Block{block: block})
			}
		} else {
			if len(labels) > 0 && labels[0] == firstLabel {
				result = append(result, &Block{block: block})
			}
		}
	}
	return result
}

// Labels returns the labels of the block.
func (b *Block) Labels() []string {
	return b.block.Labels()
}

// SetLabels replaces the labels of the block.
func (b *Block) SetLabels(labels []string) {
	b.block.SetLabels(labels)
}

// Type returns the type name of the block (e.g. "resource", "data").
func (b *Block) Type() string {
	return b.block.Type()
}

// BlockAtPath navigates nested blocks using a cty.Path, where each
// cty.GetAttrStep matches a child block by type name. Returns nil if any
// step along the path is not found or uses an unsupported step type.
// This is consistent with configschema.Block.BlockByPath.
func (b *Block) BlockAtPath(path cty.Path) *Block {
	current := b.block.Body()
	var lastChild *hclwrite.Block
	for _, step := range path {
		attrStep, ok := step.(cty.GetAttrStep)
		if !ok {
			return nil
		}
		found := false
		for _, child := range current.Blocks() {
			if child.Type() == attrStep.Name {
				lastChild = child
				current = child.Body()
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	if lastChild == nil {
		return nil
	}
	return &Block{block: lastChild}
}

// RenameAttribute renames an attribute from "from" to "to" within the block.
// Returns true if the attribute was found and renamed, false otherwise.
func (b *Block) RenameAttribute(from, to string) bool {
	return b.block.Body().RenameAttribute(from, to)
}

// RemoveAttribute removes the named attribute from the block.
// Returns true if the attribute existed and was removed, false otherwise.
func (b *Block) RemoveAttribute(name string) bool {
	return b.block.Body().RemoveAttribute(name) != nil
}

// SetAttributeValue sets (or creates) an attribute with the given cty value.
func (b *Block) SetAttributeValue(name string, val cty.Value) {
	b.block.Body().SetAttributeValue(name, val)
}

// SetAttributeRaw sets (or creates) an attribute with raw HCL tokens.
func (b *Block) SetAttributeRaw(name string, tokens hclwrite.Tokens) {
	b.block.Body().SetAttributeRaw(name, tokens)
}

// RemoveBlock removes the first nested block matching blockType.
// Returns true if a matching block was found and removed, false otherwise.
func (b *Block) RemoveBlock(blockType string) bool {
	body := b.block.Body()
	for _, child := range body.Blocks() {
		if child.Type() == blockType {
			return body.RemoveBlock(child)
		}
	}
	return false
}

// AddBlock appends a new empty nested block of the given type
// and returns it wrapped as a *Block.
func (b *Block) AddBlock(blockType string) *Block {
	nb := b.block.Body().AppendNewBlock(blockType, nil)
	return &Block{block: nb}
}

// traversalToNames converts an hcl.Traversal to the []string format
// expected by hclwrite.Expression.RenameVariablePrefix.
func traversalToNames(t hcl.Traversal) []string {
	names := make([]string, 0, len(t))
	for _, step := range t {
		switch s := step.(type) {
		case hcl.TraverseRoot:
			names = append(names, s.Name)
		case hcl.TraverseAttr:
			names = append(names, s.Name)
		}
	}
	return names
}

// renameReferencesInBody walks all attributes in a body and its nested
// blocks, calling RenameVariablePrefix on each expression.
func renameReferencesInBody(body *hclwrite.Body, old, new []string) {
	for _, attr := range body.Attributes() {
		attr.Expr().RenameVariablePrefix(old, new)
	}
	for _, block := range body.Blocks() {
		renameReferencesInBody(block.Body(), old, new)
	}
}

// RenameReferencePrefix renames variable reference prefixes within the
// block and all nested blocks. For example, renaming a traversal rooted
// at aws_s3_bucket to aws_bucket changes aws_s3_bucket.main.id to
// aws_bucket.main.id.
func (b *Block) RenameReferencePrefix(old, new hcl.Traversal) {
	renameReferencesInBody(b.block.Body(), traversalToNames(old), traversalToNames(new))
}

// RenameBlockType changes the type of all top-level blocks matching oldType
// to newType.
func (f *File) RenameBlockType(oldType, newType string) {
	for _, block := range f.file.Body().Blocks() {
		if block.Type() == oldType {
			block.SetType(newType)
		}
	}
}

// RemoveBlock removes the first top-level block matching blockType and labels.
// Returns true if a matching block was found and removed, false otherwise.
func (f *File) RemoveBlock(blockType string, labels []string) bool {
	body := f.file.Body()
	block := body.FirstMatchingBlock(blockType, labels)
	if block == nil {
		return false
	}
	return body.RemoveBlock(block)
}

// AddBlock appends a new top-level block with the given type and labels
// and returns it wrapped as a *Block.
func (f *File) AddBlock(blockType string, labels []string) *Block {
	b := f.file.Body().AppendNewBlock(blockType, labels)
	return &Block{block: b}
}

// RenameReferencePrefix renames variable reference prefixes across
// every block and attribute in the entire file.
func (f *File) RenameReferencePrefix(old, new hcl.Traversal) {
	renameReferencesInBody(f.file.Body(), traversalToNames(old), traversalToNames(new))
}
