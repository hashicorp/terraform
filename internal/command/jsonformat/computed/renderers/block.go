package renderers

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/plans"
)

var (
	_ computed.DiffRenderer = (*blockRenderer)(nil)

	importantAttributes = []string{
		"id",
	}
)

func importantAttribute(attr string) bool {
	for _, attribute := range importantAttributes {
		if attribute == attr {
			return true
		}
	}
	return false
}

func Block(attributes map[string]computed.Diff, blocks Blocks) computed.DiffRenderer {
	return &blockRenderer{
		attributes: attributes,
		blocks:     blocks,
	}
}

type blockRenderer struct {
	NoWarningsRenderer

	attributes map[string]computed.Diff
	blocks     Blocks
}

func (renderer blockRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	unchangedAttributes := 0
	unchangedBlocks := 0

	maximumAttributeKeyLen := 0
	var attributeKeys []string
	escapedAttributeKeys := make(map[string]string)
	for key := range renderer.attributes {
		attributeKeys = append(attributeKeys, key)
		escapedKey := ensureValidAttributeName(key)
		escapedAttributeKeys[key] = escapedKey
		if maximumAttributeKeyLen < len(escapedKey) {
			maximumAttributeKeyLen = len(escapedKey)
		}
	}
	sort.Strings(attributeKeys)

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("{%s\n", forcesReplacement(diff.Replace, opts.OverrideForcesReplacement)))
	for _, importantKey := range importantAttributes {
		if attribute, ok := renderer.attributes[importantKey]; ok {
			buf.WriteString(fmt.Sprintf("%s%s %-*s = %s\n", formatIndent(indent+1), format.DiffActionSymbol(attribute.Action), maximumAttributeKeyLen, importantKey, attribute.RenderHuman(indent+1, opts)))
		}
	}

	for _, key := range attributeKeys {
		if importantAttribute(key) {
			continue
		}
		attribute := renderer.attributes[key]
		if attribute.Action == plans.NoOp && !opts.ShowUnchangedChildren {
			unchangedAttributes++
			continue
		}

		for _, warning := range attribute.WarningsHuman(indent + 1) {
			buf.WriteString(fmt.Sprintf("%s%s\n", formatIndent(indent+1), warning))
		}
		buf.WriteString(fmt.Sprintf("%s%s %-*s = %s\n", formatIndent(indent+1), format.DiffActionSymbol(attribute.Action), maximumAttributeKeyLen, escapedAttributeKeys[key], attribute.RenderHuman(indent+1, opts)))
	}

	if unchangedAttributes > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", formatIndent(indent+1), format.DiffActionSymbol(plans.NoOp), unchanged("attribute", unchangedAttributes)))
	}

	blockKeys := renderer.blocks.GetAllKeys()
	for _, key := range blockKeys {

		foundChangedBlock := false
		renderBlock := func(diff computed.Diff, mapKey string, opts computed.RenderHumanOpts) {
			if diff.Action == plans.NoOp && !opts.ShowUnchangedChildren {
				unchangedBlocks++
				return
			}

			if !foundChangedBlock && len(renderer.attributes) > 0 {
				// We always want to put an extra new line between the
				// attributes and blocks, and between groups of blocks.
				buf.WriteString("\n")
				foundChangedBlock = true
			}

			for _, warning := range diff.WarningsHuman(indent + 1) {
				buf.WriteString(fmt.Sprintf("%s%s\n", formatIndent(indent+1), warning))
			}
			buf.WriteString(fmt.Sprintf("%s%s %s%s %s\n", formatIndent(indent+1), format.DiffActionSymbol(diff.Action), ensureValidAttributeName(key), mapKey, diff.RenderHuman(indent+1, opts)))

		}

		switch {
		case renderer.blocks.IsSingleBlock(key):
			renderBlock(renderer.blocks.SingleBlocks[key], "", opts)
		case renderer.blocks.IsMapBlock(key):
			var keys []string
			for key := range renderer.blocks.MapBlocks[key] {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			for _, innerKey := range keys {
				renderBlock(renderer.blocks.MapBlocks[key][innerKey], fmt.Sprintf(" %q", innerKey), opts)
			}
		case renderer.blocks.IsSetBlock(key):

			setOpts := opts.Clone()
			setOpts.OverrideForcesReplacement = diff.Replace

			for _, block := range renderer.blocks.SetBlocks[key] {
				renderBlock(block, "", opts)
			}
		case renderer.blocks.IsListBlock(key):
			for _, block := range renderer.blocks.ListBlocks[key] {
				renderBlock(block, "", opts)
			}
		}
	}

	if unchangedBlocks > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", formatIndent(indent+1), format.DiffActionSymbol(plans.NoOp), unchanged("block", unchangedBlocks)))
	}

	buf.WriteString(fmt.Sprintf("%s%s }", formatIndent(indent), format.DiffActionSymbol(plans.NoOp)))
	return buf.String()
}
