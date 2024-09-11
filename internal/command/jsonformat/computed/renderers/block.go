// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package renderers

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/plans"
)

var (
	_ computed.DiffRenderer = (*blockRenderer)(nil)

	importantAttributes = []string{
		"id",
		"name",
		"tags",
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
	if len(renderer.attributes) == 0 && len(renderer.blocks.GetAllKeys()) == 0 {
		return fmt.Sprintf("{}%s", forcesReplacement(diff.Replace, opts))
	}

	unchangedAttributes := 0
	unchangedBlocks := 0

	maximumAttributeKeyLen := 0
	var attributeKeys []string
	escapedAttributeKeys := make(map[string]string)
	for key := range renderer.attributes {
		attributeKeys = append(attributeKeys, key)
		escapedKey := EnsureValidAttributeName(key)
		escapedAttributeKeys[key] = escapedKey
		if maximumAttributeKeyLen < len(escapedKey) {
			maximumAttributeKeyLen = len(escapedKey)
		}
	}
	sort.Strings(attributeKeys)

	importantAttributeOpts := opts.Clone()
	importantAttributeOpts.ShowUnchangedChildren = true

	attributeOpts := opts.Clone()

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("{%s\n", forcesReplacement(diff.Replace, opts)))
	for _, key := range attributeKeys {
		attribute := renderer.attributes[key]
		if importantAttribute(key) {

			// Always display the important attributes.
			for _, warning := range attribute.WarningsHuman(indent+1, importantAttributeOpts) {
				buf.WriteString(fmt.Sprintf("%s%s\n", formatIndent(indent+1), warning))
			}
			buf.WriteString(fmt.Sprintf("%s%s%-*s = %s\n", formatIndent(indent+1), writeDiffActionSymbol(attribute.Action, importantAttributeOpts), maximumAttributeKeyLen, key, attribute.RenderHuman(indent+1, importantAttributeOpts)))
			continue
		}
		if attribute.Action == plans.NoOp && !opts.ShowUnchangedChildren {
			unchangedAttributes++
			continue
		}

		for _, warning := range attribute.WarningsHuman(indent+1, opts) {
			buf.WriteString(fmt.Sprintf("%s%s\n", formatIndent(indent+1), warning))
		}
		buf.WriteString(fmt.Sprintf("%s%s%-*s = %s\n", formatIndent(indent+1), writeDiffActionSymbol(attribute.Action, attributeOpts), maximumAttributeKeyLen, escapedAttributeKeys[key], attribute.RenderHuman(indent+1, attributeOpts)))
	}

	if unchangedAttributes > 0 {
		buf.WriteString(fmt.Sprintf("%s%s%s\n", formatIndent(indent+1), writeDiffActionSymbol(plans.NoOp, opts), unchanged("attribute", unchangedAttributes, opts)))
	}

	blockKeys := renderer.blocks.GetAllKeys()
	for _, key := range blockKeys {

		foundChangedBlock := false
		renderBlock := func(diff computed.Diff, mapKey string, opts computed.RenderHumanOpts) {

			creatingSensitiveValue := diff.Action == plans.Create && renderer.blocks.AfterSensitiveBlocks[key]
			deletingSensitiveValue := diff.Action == plans.Delete && renderer.blocks.BeforeSensitiveBlocks[key]
			modifyingSensitiveValue := (diff.Action == plans.Update || diff.Action == plans.NoOp) && (renderer.blocks.AfterSensitiveBlocks[key] || renderer.blocks.BeforeSensitiveBlocks[key])

			if creatingSensitiveValue || deletingSensitiveValue || modifyingSensitiveValue {
				// Intercept the renderer here if the sensitive data was set
				// across all the blocks instead of individually.
				action := diff.Action
				if diff.Action == plans.NoOp && renderer.blocks.BeforeSensitiveBlocks[key] != renderer.blocks.AfterSensitiveBlocks[key] {
					action = plans.Update
				}

				diff = computed.NewDiff(SensitiveBlock(diff, renderer.blocks.BeforeSensitiveBlocks[key], renderer.blocks.AfterSensitiveBlocks[key]), action, diff.Replace)
			}

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

			// If the force replacement metadata was set for every entry in the
			// block we need to override that here. Our child blocks will only
			// know about the replace function if it was set on them
			// specifically, and not if it was set for all the blocks.
			blockOpts := opts.Clone()
			blockOpts.ForceForcesReplacement = renderer.blocks.ReplaceBlocks[key]

			for _, warning := range diff.WarningsHuman(indent+1, blockOpts) {
				buf.WriteString(fmt.Sprintf("%s%s\n", formatIndent(indent+1), warning))
			}
			buf.WriteString(fmt.Sprintf("%s%s%s%s %s\n", formatIndent(indent+1), writeDiffActionSymbol(diff.Action, blockOpts), EnsureValidAttributeName(key), mapKey, diff.RenderHuman(indent+1, blockOpts)))

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

			if renderer.blocks.UnknownBlocks[key] {
				renderBlock(computed.NewDiff(Unknown(computed.Diff{}), diff.Action, false), "", opts)
			}

			for _, innerKey := range keys {
				renderBlock(renderer.blocks.MapBlocks[key][innerKey], fmt.Sprintf(" %q", innerKey), opts)
			}
		case renderer.blocks.IsSetBlock(key):

			setOpts := opts.Clone()
			setOpts.ForceForcesReplacement = diff.Replace

			if renderer.blocks.UnknownBlocks[key] {
				renderBlock(computed.NewDiff(Unknown(computed.Diff{}), diff.Action, false), "", opts)
			}

			for _, block := range renderer.blocks.SetBlocks[key] {
				renderBlock(block, "", opts)
			}
		case renderer.blocks.IsListBlock(key):

			if renderer.blocks.UnknownBlocks[key] {
				renderBlock(computed.NewDiff(Unknown(computed.Diff{}), diff.Action, false), "", opts)
			}

			for _, block := range renderer.blocks.ListBlocks[key] {
				renderBlock(block, "", opts)
			}
		}
	}

	if unchangedBlocks > 0 {
		buf.WriteString(fmt.Sprintf("\n%s%s%s\n", formatIndent(indent+1), writeDiffActionSymbol(plans.NoOp, opts), unchanged("block", unchangedBlocks, opts)))
	}

	buf.WriteString(fmt.Sprintf("%s%s}", formatIndent(indent), writeDiffActionSymbol(plans.NoOp, opts)))
	return buf.String()
}
