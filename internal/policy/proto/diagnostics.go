// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package proto

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
)

func ToHCLDiagnostics(diagnostics []*Diagnostic) hcl.Diagnostics {
	var diags hcl.Diagnostics
	for _, diag := range diagnostics {
		diags = diags.Append(diag.ToHCL())
	}
	return diags
}

func (diagnostic *Diagnostic) ToHCL() *hcl.Diagnostic {

	// every diagnostic we make will be identified as a "policy" diagnostic, and
	// we might add extra metadata as well.
	var extra any

	if diagnostic.PolicySet != nil {
		extra = &PolicyExtra{
			PolicySet: PolicySetMeta{
				Name: diagnostic.PolicySet.Name,
				Path: diagnostic.PolicySet.Path,
			},
		}
	} else {
		extra = new(PolicyExtra)
	}

	if diagnostic.Result != nil {
		extra = &EvaluateResultExtra{
			diagnosticExtra: diagnosticExtra{
				next: extra,
			},
			EvaluateResult: diagnostic.Result.Result,
		}
	}

	if diagnostic.Snippet != nil {
		extra = &SnippetExtra{
			diagnosticExtra: diagnosticExtra{
				next: extra,
			},
			Snippet: diagnostic.Snippet,
		}
	}

	if len(diagnostic.ExpressionValues) > 0 {
		extra = &ExpressionValuesExtra{
			diagnosticExtra: diagnosticExtra{
				next: extra,
			},
			ExpressionValues: diagnostic.ExpressionValues,
		}
	}

	if len(diagnostic.FunctionCall) > 0 {
		extra = &FunctionCallExtra{
			diagnosticExtra: diagnosticExtra{
				next: extra,
			},
			FunctionCall: diagnostic.FunctionCall,
		}
	}

	diag := &hcl.Diagnostic{
		Severity: diagnostic.Severity.ToHclSeverity(),
		Summary:  diagnostic.Summary,
		Detail:   diagnostic.Detail,
		Extra:    extra,
	}

	if diagnostic.Context != nil && diag.Subject == nil {
		// only set the context if the local range wasn't used.
		diag.Context = diagnostic.Context.ToHclRange().Ptr()
	}

	if diagnostic.Subject != nil {

		// whatever's happened, we'll record the subject and context of the
		// original diagnostic in an extra.
		diag.Extra = &RangeExtra{
			diagnosticExtra: diagnosticExtra{
				next: diag.Extra,
			},
			Subject: diagnostic.Subject,
			Context: diagnostic.Context,
		}
	}

	if diagnostic.Attribute != nil {
		attribute, err := diagnostic.Attribute.ToCtyPath()
		if err == nil {
			diag.Extra = &AttributeExtra{
				diagnosticExtra: diagnosticExtra{
					next: diag.Extra,
				},
				Attribute: attribute,
			}
		}

		// otherwise, we'll just render a diagnostic with slightly less
		// information, no big deal
	}

	return diag
}

func (severity Severity) ToHclSeverity() hcl.DiagnosticSeverity {
	switch severity {
	case Severity_ERROR:
		return hcl.DiagError
	case Severity_WARNING:
		return hcl.DiagWarning
	default:
		return hcl.DiagInvalid
	}
}

func (rng *Range) ToHclRange() hcl.Range {
	if rng == nil {
		return hcl.Range{}
	}
	return hcl.Range{
		Filename: rng.Filename,
		Start:    rng.Start.ToHclPos(),
		End:      rng.End.ToHclPos(),
	}
}

func (pos *Position) ToHclPos() hcl.Pos {
	return hcl.Pos{
		Byte:   int(pos.Byte),
		Line:   int(pos.Line),
		Column: int(pos.Column),
	}
}

// ToCtyPath converts a Path to a cty.Path.
func (path *AttributePath) ToCtyPath() (cty.Path, error) {
	var steps []cty.PathStep
	for _, step := range path.Steps {
		s, err := step.ToCtyPathStep()
		if err != nil {
			return nil, err
		}
		steps = append(steps, s)
	}
	return steps, nil
}

// ToCtyPathStep converts a Step to a cty.PathStep.
func (step *AttributePath_Step) ToCtyPathStep() (cty.PathStep, error) {
	switch step := step.Step.(type) {
	case *AttributePath_Step_Attribute:
		return cty.GetAttrStep{
			Name: step.Attribute,
		}, nil
	case *AttributePath_Step_Index:
		index, err := msgpack.Unmarshal(step.Index, cty.DynamicPseudoType)
		if err != nil {
			return nil, err
		}

		return cty.IndexStep{
			Key: index,
		}, nil
	default:
		panic(fmt.Errorf("unsupported Step type: %T", step))
	}
}
