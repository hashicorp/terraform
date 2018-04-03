package dynblock

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

type expandSpec struct {
	blockType      string
	blockTypeRange hcl.Range
	defRange       hcl.Range
	forEachVal     cty.Value
	iteratorName   string
	labelExprs     []hcl.Expression
	contentBody    hcl.Body
	inherited      map[string]*iteration
}

func (b *expandBody) decodeSpec(blockS *hcl.BlockHeaderSchema, rawSpec *hcl.Block) (*expandSpec, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	var schema *hcl.BodySchema
	if len(blockS.LabelNames) != 0 {
		schema = dynamicBlockBodySchemaLabels
	} else {
		schema = dynamicBlockBodySchemaNoLabels
	}

	specContent, specDiags := rawSpec.Body.Content(schema)
	diags = append(diags, specDiags...)
	if specDiags.HasErrors() {
		return nil, diags
	}

	//// for_each attribute

	eachAttr := specContent.Attributes["for_each"]
	eachVal, eachDiags := eachAttr.Expr.Value(b.forEachCtx)
	diags = append(diags, eachDiags...)

	if !eachVal.CanIterateElements() {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid dynamic for_each value",
			Detail:   fmt.Sprintf("Cannot use a value of type %s in for_each. An iterable collection is required.", eachVal.Type()),
			Subject:  eachAttr.Expr.Range().Ptr(),
		})
		return nil, diags
	}
	if eachVal.IsNull() {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid dynamic for_each value",
			Detail:   "Cannot use a null value in for_each.",
			Subject:  eachAttr.Expr.Range().Ptr(),
		})
		return nil, diags
	}

	//// iterator attribute

	iteratorName := blockS.Type
	if iteratorAttr := specContent.Attributes["iterator"]; iteratorAttr != nil {
		itTraversal, itDiags := hcl.AbsTraversalForExpr(iteratorAttr.Expr)
		diags = append(diags, itDiags...)
		if itDiags.HasErrors() {
			return nil, diags
		}

		if len(itTraversal) != 1 {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid dynamic iterator name",
				Detail:   "Dynamic iterator must be a single variable name.",
				Subject:  itTraversal.SourceRange().Ptr(),
			})
			return nil, diags
		}

		iteratorName = itTraversal.RootName()
	}

	var labelExprs []hcl.Expression
	if labelsAttr := specContent.Attributes["labels"]; labelsAttr != nil {
		var labelDiags hcl.Diagnostics
		labelExprs, labelDiags = hcl.ExprList(labelsAttr.Expr)
		diags = append(diags, labelDiags...)
		if labelDiags.HasErrors() {
			return nil, diags
		}

		if len(labelExprs) > len(blockS.LabelNames) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Extraneous dynamic block label",
				Detail:   fmt.Sprintf("Blocks of type %q require %d label(s).", blockS.Type, len(blockS.LabelNames)),
				Subject:  labelExprs[len(blockS.LabelNames)].Range().Ptr(),
			})
			return nil, diags
		} else if len(labelExprs) < len(blockS.LabelNames) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Insufficient dynamic block labels",
				Detail:   fmt.Sprintf("Blocks of type %q require %d label(s).", blockS.Type, len(blockS.LabelNames)),
				Subject:  labelsAttr.Expr.Range().Ptr(),
			})
			return nil, diags
		}
	}

	// Since our schema requests only blocks of type "content", we can assume
	// that all entries in specContent.Blocks are content blocks.
	if len(specContent.Blocks) == 0 {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing dynamic content block",
			Detail:   "A dynamic block must have a nested block of type \"content\" to describe the body of each generated block.",
			Subject:  &specContent.MissingItemRange,
		})
		return nil, diags
	}
	if len(specContent.Blocks) > 1 {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Extraneous dynamic content block",
			Detail:   "Only one nested content block is allowed for each dynamic block.",
			Subject:  &specContent.Blocks[1].DefRange,
		})
		return nil, diags
	}

	return &expandSpec{
		blockType:      blockS.Type,
		blockTypeRange: rawSpec.LabelRanges[0],
		defRange:       rawSpec.DefRange,
		forEachVal:     eachVal,
		iteratorName:   iteratorName,
		labelExprs:     labelExprs,
		contentBody:    specContent.Blocks[0].Body,
	}, diags
}

func (s *expandSpec) newBlock(i *iteration, ctx *hcl.EvalContext) (*hcl.Block, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var labels []string
	var labelRanges []hcl.Range
	lCtx := i.EvalContext(ctx)
	for _, labelExpr := range s.labelExprs {
		labelVal, labelDiags := labelExpr.Value(lCtx)
		diags = append(diags, labelDiags...)
		if labelDiags.HasErrors() {
			return nil, diags
		}

		var convErr error
		labelVal, convErr = convert.Convert(labelVal, cty.String)
		if convErr != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid dynamic block label",
				Detail:   fmt.Sprintf("Cannot use this value as a dynamic block label: %s.", convErr),
				Subject:  labelExpr.Range().Ptr(),
			})
			return nil, diags
		}
		if labelVal.IsNull() {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid dynamic block label",
				Detail:   "Cannot use a null value as a dynamic block label.",
				Subject:  labelExpr.Range().Ptr(),
			})
			return nil, diags
		}
		if !labelVal.IsKnown() {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid dynamic block label",
				Detail:   "This value is not yet known. Dynamic block labels must be immediately-known values.",
				Subject:  labelExpr.Range().Ptr(),
			})
			return nil, diags
		}

		labels = append(labels, labelVal.AsString())
		labelRanges = append(labelRanges, labelExpr.Range())
	}

	block := &hcl.Block{
		Type:        s.blockType,
		TypeRange:   s.blockTypeRange,
		Labels:      labels,
		LabelRanges: labelRanges,
		DefRange:    s.defRange,
		Body:        s.contentBody,
	}

	return block, diags
}
