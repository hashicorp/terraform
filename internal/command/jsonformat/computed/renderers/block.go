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

func Block(attributes map[string]computed.Diff, blocks map[string][]computed.Diff) computed.DiffRenderer {
	return &blockRenderer{
		attributes: attributes,
		blocks:     blocks,
	}
}

type blockRenderer struct {
	NoWarningsRenderer

	attributes map[string]computed.Diff
	blocks     map[string][]computed.Diff
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
	buf.WriteString(fmt.Sprintf("{%s\n", forcesReplacement(diff.Replace)))
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

	var blockKeys []string
	for key := range renderer.blocks {
		blockKeys = append(blockKeys, key)
	}
	sort.Strings(blockKeys)

	for _, key := range blockKeys {
		blocks := renderer.blocks[key]

		foundChangedBlock := false
		for _, block := range blocks {
			if block.Action == plans.NoOp && !opts.ShowUnchangedChildren {
				unchangedBlocks++
				continue
			}

			if !foundChangedBlock && len(renderer.attributes) > 0 {
				buf.WriteString("\n")
				foundChangedBlock = true
			}

			for _, warning := range block.WarningsHuman(indent + 1) {
				buf.WriteString(fmt.Sprintf("%s%s\n", formatIndent(indent+1), warning))
			}
			buf.WriteString(fmt.Sprintf("%s%s %s %s\n", formatIndent(indent+1), format.DiffActionSymbol(block.Action), ensureValidAttributeName(key), block.RenderHuman(indent+1, opts)))
		}
	}

	if unchangedBlocks > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", formatIndent(indent+1), format.DiffActionSymbol(plans.NoOp), unchanged("block", unchangedBlocks)))
	}

	buf.WriteString(fmt.Sprintf("%s%s }", formatIndent(indent), format.DiffActionSymbol(plans.NoOp)))
	return buf.String()
}
