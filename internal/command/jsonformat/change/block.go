package change

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/plans"
)

var (
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

func Block(attributes map[string]Change, blocks map[string][]Change) Renderer {
	return &blockRenderer{
		attributes: attributes,
		blocks:     blocks,
	}
}

type blockRenderer struct {
	NoWarningsRenderer

	attributes map[string]Change
	blocks     map[string][]Change
}

func (renderer blockRenderer) Render(change Change, indent int, opts RenderOpts) string {
	unchangedAttributes := 0
	unchangedBlocks := 0

	maximumAttributeKeyLen := 0
	var attributeKeys []string
	escapedAttributeKeys := make(map[string]string)
	for key := range renderer.attributes {
		attributeKeys = append(attributeKeys, key)
		escapedKey := change.ensureValidAttributeName(key)
		escapedAttributeKeys[key] = escapedKey
		if maximumAttributeKeyLen < len(escapedKey) {
			maximumAttributeKeyLen = len(escapedKey)
		}
	}
	sort.Strings(attributeKeys)

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("{%s\n", change.forcesReplacement()))
	for _, importantKey := range importantAttributes {
		if attribute, ok := renderer.attributes[importantKey]; ok {
			buf.WriteString(fmt.Sprintf("%s%s %-*s = %s\n", change.indent(indent+1), format.DiffActionSymbol(attribute.action), maximumAttributeKeyLen, importantKey, attribute.Render(indent+1, opts)))
		}
	}

	for _, key := range attributeKeys {
		if importantAttribute(key) {
			continue
		}
		attribute := renderer.attributes[key]
		if attribute.action == plans.NoOp && !opts.showUnchangedChildren {
			unchangedAttributes++
			continue
		}

		for _, warning := range attribute.Warnings(indent + 1) {
			buf.WriteString(fmt.Sprintf("%s%s\n", change.indent(indent+1), warning))
		}
		buf.WriteString(fmt.Sprintf("%s%s %-*s = %s\n", change.indent(indent+1), format.DiffActionSymbol(attribute.action), maximumAttributeKeyLen, escapedAttributeKeys[key], attribute.Render(indent+1, opts)))
	}

	if unchangedAttributes > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), format.DiffActionSymbol(plans.NoOp), change.unchanged("attribute", unchangedAttributes)))
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
			if block.action == plans.NoOp && !opts.showUnchangedChildren {
				unchangedBlocks++
				continue
			}

			if !foundChangedBlock && len(renderer.attributes) > 0 {
				buf.WriteString("\n")
				foundChangedBlock = true
			}

			for _, warning := range block.Warnings(indent + 1) {
				buf.WriteString(fmt.Sprintf("%s%s\n", change.indent(indent+1), warning))
			}
			buf.WriteString(fmt.Sprintf("%s%s %s %s\n", change.indent(indent+1), format.DiffActionSymbol(block.action), change.ensureValidAttributeName(key), block.Render(indent+1, opts)))
		}
	}

	if unchangedBlocks > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), format.DiffActionSymbol(plans.NoOp), change.unchanged("block", unchangedBlocks)))
	}

	buf.WriteString(fmt.Sprintf("%s%s }", change.indent(indent), format.DiffActionSymbol(plans.NoOp)))
	return buf.String()
}
