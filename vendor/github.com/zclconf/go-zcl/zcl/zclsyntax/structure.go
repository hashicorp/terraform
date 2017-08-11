package zclsyntax

import (
	"fmt"
	"strings"

	"github.com/zclconf/go-zcl/zcl"
)

// AsZCLBlock returns the block data expressed as a *zcl.Block.
func (b *Block) AsZCLBlock() *zcl.Block {
	lastHeaderRange := b.TypeRange
	if len(b.LabelRanges) > 0 {
		lastHeaderRange = b.LabelRanges[len(b.LabelRanges)-1]
	}

	return &zcl.Block{
		Type:   b.Type,
		Labels: b.Labels,
		Body:   b.Body,

		DefRange:    zcl.RangeBetween(b.TypeRange, lastHeaderRange),
		TypeRange:   b.TypeRange,
		LabelRanges: b.LabelRanges,
	}
}

// Body is the implementation of zcl.Body for the zcl native syntax.
type Body struct {
	Attributes Attributes
	Blocks     Blocks

	// These are used with PartialContent to produce a "remaining items"
	// body to return. They are nil on all bodies fresh out of the parser.
	hiddenAttrs  map[string]struct{}
	hiddenBlocks map[string]struct{}

	SrcRange zcl.Range
	EndRange zcl.Range // Final token of the body, for reporting missing items
}

// Assert that *Body implements zcl.Body
var assertBodyImplBody zcl.Body = &Body{}

func (b *Body) walkChildNodes(w internalWalkFunc) {
	b.Attributes = w(b.Attributes).(Attributes)
	b.Blocks = w(b.Blocks).(Blocks)
}

func (b *Body) Range() zcl.Range {
	return b.SrcRange
}

func (b *Body) Content(schema *zcl.BodySchema) (*zcl.BodyContent, zcl.Diagnostics) {
	content, remainZCL, diags := b.PartialContent(schema)

	// No we'll see if anything actually remains, to produce errors about
	// extraneous items.
	remain := remainZCL.(*Body)

	for name, attr := range b.Attributes {
		if _, hidden := remain.hiddenAttrs[name]; !hidden {
			var suggestions []string
			for _, attrS := range schema.Attributes {
				if _, defined := content.Attributes[attrS.Name]; defined {
					continue
				}
				suggestions = append(suggestions, attrS.Name)
			}
			suggestion := nameSuggestion(name, suggestions)
			if suggestion != "" {
				suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
			} else {
				// Is there a block of the same name?
				for _, blockS := range schema.Blocks {
					if blockS.Type == name {
						suggestion = fmt.Sprintf(" Did you mean to define a block of type %q?", name)
						break
					}
				}
			}

			diags = append(diags, &zcl.Diagnostic{
				Severity: zcl.DiagError,
				Summary:  "Unsupported attribute",
				Detail:   fmt.Sprintf("An attribute named %q is not expected here.%s", name, suggestion),
				Subject:  &attr.NameRange,
			})
		}
	}

	for _, block := range b.Blocks {
		blockTy := block.Type
		if _, hidden := remain.hiddenBlocks[blockTy]; !hidden {
			var suggestions []string
			for _, blockS := range schema.Blocks {
				suggestions = append(suggestions, blockS.Type)
			}
			suggestion := nameSuggestion(blockTy, suggestions)
			if suggestion != "" {
				suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
			} else {
				// Is there an attribute of the same name?
				for _, attrS := range schema.Attributes {
					if attrS.Name == blockTy {
						suggestion = fmt.Sprintf(" Did you mean to define attribute %q?", blockTy)
						break
					}
				}
			}

			diags = append(diags, &zcl.Diagnostic{
				Severity: zcl.DiagError,
				Summary:  "Unsupported block type",
				Detail:   fmt.Sprintf("Blocks of type %q are not expected here.%s", blockTy, suggestion),
				Subject:  &block.TypeRange,
			})
		}
	}

	return content, diags
}

func (b *Body) PartialContent(schema *zcl.BodySchema) (*zcl.BodyContent, zcl.Body, zcl.Diagnostics) {
	attrs := make(zcl.Attributes)
	var blocks zcl.Blocks
	var diags zcl.Diagnostics
	hiddenAttrs := make(map[string]struct{})
	hiddenBlocks := make(map[string]struct{})

	if b.hiddenAttrs != nil {
		for k, v := range b.hiddenAttrs {
			hiddenAttrs[k] = v
		}
	}
	if b.hiddenBlocks != nil {
		for k, v := range b.hiddenBlocks {
			hiddenBlocks[k] = v
		}
	}

	for _, attrS := range schema.Attributes {
		name := attrS.Name
		attr, exists := b.Attributes[name]
		_, hidden := hiddenAttrs[name]
		if hidden || !exists {
			if attrS.Required {
				diags = append(diags, &zcl.Diagnostic{
					Severity: zcl.DiagError,
					Summary:  "Missing required attribute",
					Detail:   fmt.Sprintf("The attribute %q is required, but no definition was found.", attrS.Name),
					Subject:  b.MissingItemRange().Ptr(),
				})
			}
			continue
		}

		hiddenAttrs[name] = struct{}{}
		attrs[name] = attr.AsZCLAttribute()
	}

	blocksWanted := make(map[string]zcl.BlockHeaderSchema)
	for _, blockS := range schema.Blocks {
		blocksWanted[blockS.Type] = blockS
	}

	for _, block := range b.Blocks {
		if _, hidden := hiddenBlocks[block.Type]; hidden {
			continue
		}
		blockS, wanted := blocksWanted[block.Type]
		if !wanted {
			continue
		}

		if len(block.Labels) > len(blockS.LabelNames) {
			name := block.Type
			if len(blockS.LabelNames) == 0 {
				diags = append(diags, &zcl.Diagnostic{
					Severity: zcl.DiagError,
					Summary:  fmt.Sprintf("Extraneous label for %s", name),
					Detail: fmt.Sprintf(
						"No labels are expected for %s blocks.", name,
					),
					Subject: block.LabelRanges[0].Ptr(),
					Context: zcl.RangeBetween(block.TypeRange, block.OpenBraceRange).Ptr(),
				})
			} else {
				diags = append(diags, &zcl.Diagnostic{
					Severity: zcl.DiagError,
					Summary:  fmt.Sprintf("Extraneous label for %s", name),
					Detail: fmt.Sprintf(
						"Only %d labels (%s) are expected for %s blocks.",
						len(blockS.LabelNames), strings.Join(blockS.LabelNames, ", "), name,
					),
					Subject: block.LabelRanges[len(blockS.LabelNames)].Ptr(),
					Context: zcl.RangeBetween(block.TypeRange, block.OpenBraceRange).Ptr(),
				})
			}
			continue
		}

		if len(block.Labels) < len(blockS.LabelNames) {
			name := block.Type
			diags = append(diags, &zcl.Diagnostic{
				Severity: zcl.DiagError,
				Summary:  fmt.Sprintf("Missing %s for %s", blockS.LabelNames[len(block.Labels)], name),
				Detail: fmt.Sprintf(
					"All %s blocks must have %d labels (%s).",
					name, len(blockS.LabelNames), strings.Join(blockS.LabelNames, ", "),
				),
				Subject: &block.OpenBraceRange,
				Context: zcl.RangeBetween(block.TypeRange, block.OpenBraceRange).Ptr(),
			})
			continue
		}

		blocks = append(blocks, block.AsZCLBlock())
	}

	// We hide blocks only after we've processed all of them, since otherwise
	// we can't process more than one of the same type.
	for _, blockS := range schema.Blocks {
		hiddenBlocks[blockS.Type] = struct{}{}
	}

	remain := &Body{
		Attributes: b.Attributes,
		Blocks:     b.Blocks,

		hiddenAttrs:  hiddenAttrs,
		hiddenBlocks: hiddenBlocks,

		SrcRange: b.SrcRange,
		EndRange: b.EndRange,
	}

	return &zcl.BodyContent{
		Attributes: attrs,
		Blocks:     blocks,

		MissingItemRange: b.MissingItemRange(),
	}, remain, diags
}

func (b *Body) JustAttributes() (zcl.Attributes, zcl.Diagnostics) {
	attrs := make(zcl.Attributes)
	var diags zcl.Diagnostics

	if len(b.Blocks) > 0 {
		example := b.Blocks[0]
		diags = append(diags, &zcl.Diagnostic{
			Severity: zcl.DiagError,
			Summary:  fmt.Sprintf("Unexpected %s block", example.Type),
			Detail:   "Blocks are not allowed here.",
			Context:  &example.TypeRange,
		})
		// we will continue processing anyway, and return the attributes
		// we are able to find so that certain analyses can still be done
		// in the face of errors.
	}

	if b.Attributes == nil {
		return attrs, diags
	}

	for name, attr := range b.Attributes {
		if _, hidden := b.hiddenAttrs[name]; hidden {
			continue
		}
		attrs[name] = attr.AsZCLAttribute()
	}

	return attrs, diags
}

func (b *Body) MissingItemRange() zcl.Range {
	return b.EndRange
}

// Attributes is the collection of attribute definitions within a body.
type Attributes map[string]*Attribute

func (a Attributes) walkChildNodes(w internalWalkFunc) {
	for k, attr := range a {
		a[k] = w(attr).(*Attribute)
	}
}

// Range returns the range of some arbitrary point within the set of
// attributes, or an invalid range if there are no attributes.
//
// This is provided only to complete the Node interface, but has no practical
// use.
func (a Attributes) Range() zcl.Range {
	// An attributes doesn't really have a useful range to report, since
	// it's just a grouping construct. So we'll arbitrarily take the
	// range of one of the attributes, or produce an invalid range if we have
	// none. In practice, there's little reason to ask for the range of
	// an Attributes.
	for _, attr := range a {
		return attr.Range()
	}
	return zcl.Range{
		Filename: "<unknown>",
	}
}

// Attribute represents a single attribute definition within a body.
type Attribute struct {
	Name string
	Expr Expression

	SrcRange    zcl.Range
	NameRange   zcl.Range
	EqualsRange zcl.Range
}

func (a *Attribute) walkChildNodes(w internalWalkFunc) {
	a.Expr = w(a.Expr).(Expression)
}

func (a *Attribute) Range() zcl.Range {
	return a.SrcRange
}

// AsZCLAttribute returns the block data expressed as a *zcl.Attribute.
func (a *Attribute) AsZCLAttribute() *zcl.Attribute {
	return &zcl.Attribute{
		Name: a.Name,
		Expr: a.Expr,

		Range:     a.SrcRange,
		NameRange: a.NameRange,
	}
}

// Blocks is the list of nested blocks within a body.
type Blocks []*Block

func (bs Blocks) walkChildNodes(w internalWalkFunc) {
	for i, block := range bs {
		bs[i] = w(block).(*Block)
	}
}

// Range returns the range of some arbitrary point within the list of
// blocks, or an invalid range if there are no blocks.
//
// This is provided only to complete the Node interface, but has no practical
// use.
func (bs Blocks) Range() zcl.Range {
	if len(bs) > 0 {
		return bs[0].Range()
	}
	return zcl.Range{
		Filename: "<unknown>",
	}
}

// Block represents a nested block structure
type Block struct {
	Type   string
	Labels []string
	Body   *Body

	TypeRange       zcl.Range
	LabelRanges     []zcl.Range
	OpenBraceRange  zcl.Range
	CloseBraceRange zcl.Range
}

func (b *Block) walkChildNodes(w internalWalkFunc) {
	b.Body = w(b.Body).(*Body)
}

func (b *Block) Range() zcl.Range {
	return zcl.RangeBetween(b.TypeRange, b.CloseBraceRange)
}
