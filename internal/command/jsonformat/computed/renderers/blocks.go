// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package renderers

import (
	"sort"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
)

// Blocks is a helper struct for collating the different kinds of blocks in a
// simple way for rendering.
type Blocks struct {
	SingleBlocks map[string]computed.Diff
	ListBlocks   map[string][]computed.Diff
	SetBlocks    map[string][]computed.Diff
	MapBlocks    map[string]map[string]computed.Diff

	// ReplaceBlocks and Before/AfterSensitiveBlocks carry forward the
	// information about an entire group of blocks (eg. if all the blocks for a
	// given list block are sensitive that isn't captured in the individual
	// blocks as they are processed independently). These maps allow the
	// renderer to check the metadata on the overall groups and respond
	// accordingly.

	ReplaceBlocks         map[string]bool
	BeforeSensitiveBlocks map[string]bool
	AfterSensitiveBlocks  map[string]bool
	UnknownBlocks         map[string]bool
}

func (blocks *Blocks) GetAllKeys() []string {
	var keys []string
	for key := range blocks.SingleBlocks {
		keys = append(keys, key)
	}
	for key := range blocks.ListBlocks {
		keys = append(keys, key)
	}
	for key := range blocks.SetBlocks {
		keys = append(keys, key)
	}
	for key := range blocks.MapBlocks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (blocks *Blocks) IsSingleBlock(key string) bool {
	_, ok := blocks.SingleBlocks[key]
	return ok
}

func (blocks *Blocks) IsListBlock(key string) bool {
	_, ok := blocks.ListBlocks[key]
	return ok
}

func (blocks *Blocks) IsMapBlock(key string) bool {
	_, ok := blocks.MapBlocks[key]
	return ok
}

func (blocks *Blocks) IsSetBlock(key string) bool {
	_, ok := blocks.SetBlocks[key]
	return ok
}

func (blocks *Blocks) AddSingleBlock(key string, diff computed.Diff, replace, beforeSensitive, afterSensitive, unknown bool) {
	blocks.SingleBlocks[key] = diff
	blocks.ReplaceBlocks[key] = replace
	blocks.BeforeSensitiveBlocks[key] = beforeSensitive
	blocks.AfterSensitiveBlocks[key] = afterSensitive
	blocks.UnknownBlocks[key] = unknown
}

func (blocks *Blocks) AddAllListBlock(key string, diffs []computed.Diff, replace, beforeSensitive, afterSensitive, unknown bool) {
	blocks.ListBlocks[key] = diffs
	blocks.ReplaceBlocks[key] = replace
	blocks.BeforeSensitiveBlocks[key] = beforeSensitive
	blocks.AfterSensitiveBlocks[key] = afterSensitive
	blocks.UnknownBlocks[key] = unknown
}

func (blocks *Blocks) AddAllSetBlock(key string, diffs []computed.Diff, replace, beforeSensitive, afterSensitive, unknown bool) {
	blocks.SetBlocks[key] = diffs
	blocks.ReplaceBlocks[key] = replace
	blocks.BeforeSensitiveBlocks[key] = beforeSensitive
	blocks.AfterSensitiveBlocks[key] = afterSensitive
	blocks.UnknownBlocks[key] = unknown
}

func (blocks *Blocks) AddAllMapBlocks(key string, diffs map[string]computed.Diff, replace, beforeSensitive, afterSensitive, unknown bool) {
	blocks.MapBlocks[key] = diffs
	blocks.ReplaceBlocks[key] = replace
	blocks.BeforeSensitiveBlocks[key] = beforeSensitive
	blocks.AfterSensitiveBlocks[key] = afterSensitive
	blocks.UnknownBlocks[key] = unknown
}
