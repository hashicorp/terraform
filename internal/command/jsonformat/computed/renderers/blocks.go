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

func (blocks *Blocks) AddSingleBlock(key string, diff computed.Diff) {
	blocks.SingleBlocks[key] = diff
}

func (blocks *Blocks) AddListBlock(key string, diff computed.Diff) {
	blocks.ListBlocks[key] = append(blocks.ListBlocks[key], diff)
}

func (blocks *Blocks) AddSetBlock(key string, diff computed.Diff) {
	blocks.SetBlocks[key] = append(blocks.SetBlocks[key], diff)
}

func (blocks *Blocks) AddMapBlock(key string, entry string, diff computed.Diff) {
	m := blocks.MapBlocks[key]
	if m == nil {
		m = make(map[string]computed.Diff)
	}
	m[entry] = diff
	blocks.MapBlocks[key] = m
}
