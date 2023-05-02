// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfdiags

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// The "contextual" family of diagnostics are designed to allow separating
// the detection of a problem from placing that problem in context. For
// example, some code that is validating an object extracted from configuration
// may not have access to the configuration that generated it, but can still
// report problems within that object which the caller can then place in
// context by calling IsConfigBody on the returned diagnostics.
//
// When contextual diagnostics are used, the documentation for a method must
// be very explicit about what context is implied for any diagnostics returned,
// to help ensure the expected result.

// contextualFromConfig is an interface type implemented by diagnostic types
// that can elaborate themselves when given information about the configuration
// body they are embedded in, as well as the runtime address associated with
// that configuration.
//
// Usually this entails extracting source location information in order to
// populate the "Subject" range.
type contextualFromConfigBody interface {
	ElaborateFromConfigBody(hcl.Body, string) Diagnostic
}

// InConfigBody returns a copy of the receiver with any config-contextual
// diagnostics elaborated in the context of the given body. An optional address
// argument may be added to indicate which instance of the configuration the
// error related to.
func (diags Diagnostics) InConfigBody(body hcl.Body, addr string) Diagnostics {
	if len(diags) == 0 {
		return nil
	}

	ret := make(Diagnostics, len(diags))
	for i, srcDiag := range diags {
		if cd, isCD := srcDiag.(contextualFromConfigBody); isCD {
			ret[i] = cd.ElaborateFromConfigBody(body, addr)
		} else {
			ret[i] = srcDiag
		}
	}

	return ret
}

// AttributeValue returns a diagnostic about an attribute value in an implied current
// configuration context. This should be returned only from functions whose
// interface specifies a clear configuration context that this will be
// resolved in.
//
// The given path is relative to the implied configuration context. To describe
// a top-level attribute, it should be a single-element cty.Path with a
// cty.GetAttrStep. It's assumed that the path is returning into a structure
// that would be produced by our conventions in the configschema package; it
// may return unexpected results for structures that can't be represented by
// configschema.
//
// Since mapping attribute paths back onto configuration is an imprecise
// operation (e.g. dynamic block generation may cause the same block to be
// evaluated multiple times) the diagnostic detail should include the attribute
// name and other context required to help the user understand what is being
// referenced in case the identified source range is not unique.
//
// The returned attribute will not have source location information until
// context is applied to the containing diagnostics using diags.InConfigBody.
// After context is applied, the source location is the value assigned to the
// named attribute, or the containing body's "missing item range" if no
// value is present.
func AttributeValue(severity Severity, summary, detail string, attrPath cty.Path) Diagnostic {
	return &attributeDiagnostic{
		diagnosticBase: diagnosticBase{
			severity: severity,
			summary:  summary,
			detail:   detail,
		},
		attrPath: attrPath,
	}
}

// GetAttribute extracts an attribute cty.Path from a diagnostic if it contains
// one. Normally this is not accessed directly, and instead the config body is
// added to the Diagnostic to create a more complete message for the user. In
// some cases however, we may want to know just the name of the attribute that
// generated the Diagnostic message.
// This returns a nil cty.Path if it does not exist in the Diagnostic.
func GetAttribute(d Diagnostic) cty.Path {
	if d, ok := d.(*attributeDiagnostic); ok {
		return d.attrPath
	}
	return nil
}

type attributeDiagnostic struct {
	diagnosticBase
	attrPath cty.Path
	subject  *SourceRange // populated only after ElaborateFromConfigBody
}

// ElaborateFromConfigBody finds the most accurate possible source location
// for a diagnostic's attribute path within the given body.
//
// Backing out from a path back to a source location is not always entirely
// possible because we lose some information in the decoding process, so
// if an exact position cannot be found then the returned diagnostic will
// refer to a position somewhere within the containing body, which is assumed
// to be better than no location at all.
//
// If possible it is generally better to report an error at a layer where
// source location information is still available, for more accuracy. This
// is not always possible due to system architecture, so this serves as a
// "best effort" fallback behavior for such situations.
func (d *attributeDiagnostic) ElaborateFromConfigBody(body hcl.Body, addr string) Diagnostic {
	// don't change an existing address
	if d.address == "" {
		d.address = addr
	}

	if len(d.attrPath) < 1 {
		// Should never happen, but we'll allow it rather than crashing.
		return d
	}

	if d.subject != nil {
		// Don't modify an already-elaborated diagnostic.
		return d
	}

	ret := *d

	// This function will often end up re-decoding values that were already
	// decoded by an earlier step. This is non-ideal but is architecturally
	// more convenient than arranging for source location information to be
	// propagated to every place in Terraform, and this happens only in the
	// presence of errors where performance isn't a concern.

	traverse := d.attrPath[:]
	final := d.attrPath[len(d.attrPath)-1]

	// Index should never be the first step
	// as indexing of top blocks (such as resources & data sources)
	// is handled elsewhere
	if _, isIdxStep := traverse[0].(cty.IndexStep); isIdxStep {
		subject := SourceRangeFromHCL(body.MissingItemRange())
		ret.subject = &subject
		return &ret
	}

	// Process index separately
	idxStep, hasIdx := final.(cty.IndexStep)
	if hasIdx {
		final = d.attrPath[len(d.attrPath)-2]
		traverse = d.attrPath[:len(d.attrPath)-1]
	}

	// If we have more than one step after removing index
	// then we'll first try to traverse to a child body
	// corresponding to the requested path.
	if len(traverse) > 1 {
		body = traversePathSteps(traverse, body)
	}

	// Default is to indicate a missing item in the deepest body we reached
	// while traversing.
	subject := SourceRangeFromHCL(body.MissingItemRange())
	ret.subject = &subject

	// Once we get here, "final" should be a GetAttr step that maps to an
	// attribute in our current body.
	finalStep, isAttr := final.(cty.GetAttrStep)
	if !isAttr {
		return &ret
	}

	content, _, contentDiags := body.PartialContent(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{
				Name:     finalStep.Name,
				Required: true,
			},
		},
	})
	if contentDiags.HasErrors() {
		return &ret
	}

	if attr, ok := content.Attributes[finalStep.Name]; ok {
		hclRange := attr.Expr.Range()
		if hasIdx {
			// Try to be more precise by finding index range
			hclRange = hclRangeFromIndexStepAndAttribute(idxStep, attr)
		}
		subject = SourceRangeFromHCL(hclRange)
		ret.subject = &subject
	}

	return &ret
}

func traversePathSteps(traverse []cty.PathStep, body hcl.Body) hcl.Body {
	for i := 0; i < len(traverse); i++ {
		step := traverse[i]

		switch tStep := step.(type) {
		case cty.GetAttrStep:

			var next cty.PathStep
			if i < (len(traverse) - 1) {
				next = traverse[i+1]
			}

			// Will be indexing into our result here?
			var indexType cty.Type
			var indexVal cty.Value
			if nextIndex, ok := next.(cty.IndexStep); ok {
				indexVal = nextIndex.Key
				indexType = indexVal.Type()
				i++ // skip over the index on subsequent iterations
			}

			var blockLabelNames []string
			if indexType == cty.String {
				// Map traversal means we expect one label for the key.
				blockLabelNames = []string{"key"}
			}

			// For intermediate steps we expect to be referring to a child
			// block, so we'll attempt decoding under that assumption.
			content, _, contentDiags := body.PartialContent(&hcl.BodySchema{
				Blocks: []hcl.BlockHeaderSchema{
					{
						Type:       tStep.Name,
						LabelNames: blockLabelNames,
					},
				},
			})
			if contentDiags.HasErrors() {
				return body
			}
			filtered := make([]*hcl.Block, 0, len(content.Blocks))
			for _, block := range content.Blocks {
				if block.Type == tStep.Name {
					filtered = append(filtered, block)
				}
			}
			if len(filtered) == 0 {
				// Step doesn't refer to a block
				continue
			}

			switch indexType {
			case cty.NilType: // no index at all
				if len(filtered) != 1 {
					return body
				}
				body = filtered[0].Body
			case cty.Number:
				var idx int
				err := gocty.FromCtyValue(indexVal, &idx)
				if err != nil || idx >= len(filtered) {
					return body
				}
				body = filtered[idx].Body
			case cty.String:
				key := indexVal.AsString()
				var block *hcl.Block
				for _, candidate := range filtered {
					if candidate.Labels[0] == key {
						block = candidate
						break
					}
				}
				if block == nil {
					// No block with this key, so we'll just indicate a
					// missing item in the containing block.
					return body
				}
				body = block.Body
			default:
				// Should never happen, because only string and numeric indices
				// are supported by cty collections.
				return body
			}

		default:
			// For any other kind of step, we'll just return our current body
			// as the subject and accept that this is a little inaccurate.
			return body
		}
	}
	return body
}

func hclRangeFromIndexStepAndAttribute(idxStep cty.IndexStep, attr *hcl.Attribute) hcl.Range {
	switch idxStep.Key.Type() {
	case cty.Number:
		var idx int
		err := gocty.FromCtyValue(idxStep.Key, &idx)
		items, diags := hcl.ExprList(attr.Expr)
		if diags.HasErrors() {
			return attr.Expr.Range()
		}
		if err != nil || idx >= len(items) {
			return attr.NameRange
		}
		return items[idx].Range()
	case cty.String:
		pairs, diags := hcl.ExprMap(attr.Expr)
		if diags.HasErrors() {
			return attr.Expr.Range()
		}
		stepKey := idxStep.Key.AsString()
		for _, kvPair := range pairs {
			key, diags := kvPair.Key.Value(nil)
			if diags.HasErrors() {
				return attr.Expr.Range()
			}
			if key.AsString() == stepKey {
				startRng := kvPair.Value.StartRange()
				return startRng
			}
		}
		return attr.NameRange
	}
	return attr.Expr.Range()
}

func (d *attributeDiagnostic) Source() Source {
	return Source{
		Subject: d.subject,
	}
}

// WholeContainingBody returns a diagnostic about the body that is an implied
// current configuration context. This should be returned only from
// functions whose interface specifies a clear configuration context that this
// will be resolved in.
//
// The returned attribute will not have source location information until
// context is applied to the containing diagnostics using diags.InConfigBody.
// After context is applied, the source location is currently the missing item
// range of the body. In future, this may change to some other suitable
// part of the containing body.
func WholeContainingBody(severity Severity, summary, detail string) Diagnostic {
	return &wholeBodyDiagnostic{
		diagnosticBase: diagnosticBase{
			severity: severity,
			summary:  summary,
			detail:   detail,
		},
	}
}

type wholeBodyDiagnostic struct {
	diagnosticBase
	subject *SourceRange // populated only after ElaborateFromConfigBody
}

func (d *wholeBodyDiagnostic) ElaborateFromConfigBody(body hcl.Body, addr string) Diagnostic {
	// don't change an existing address
	if d.address == "" {
		d.address = addr
	}

	if d.subject != nil {
		// Don't modify an already-elaborated diagnostic.
		return d
	}

	ret := *d
	rng := SourceRangeFromHCL(body.MissingItemRange())
	ret.subject = &rng
	return &ret
}

func (d *wholeBodyDiagnostic) Source() Source {
	return Source{
		Subject: d.subject,
	}
}
