// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/zclconf/go-cty/cty"
	msgpack "github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func diagnosticsToProto(diags tfdiags.Diagnostics) []*terraform1.Diagnostic {
	if len(diags) == 0 {
		return nil
	}

	ret := make([]*terraform1.Diagnostic, len(diags))
	for i, diag := range diags {
		ret[i] = diagnosticToProto(diag)
	}
	return ret
}

func diagnosticToProto(diag tfdiags.Diagnostic) *terraform1.Diagnostic {
	protoDiag := &terraform1.Diagnostic{}

	switch diag.Severity() {
	case tfdiags.Error:
		protoDiag.Severity = terraform1.Diagnostic_ERROR
	case tfdiags.Warning:
		protoDiag.Severity = terraform1.Diagnostic_WARNING
	default:
		protoDiag.Severity = terraform1.Diagnostic_INVALID
	}

	desc := diag.Description()
	protoDiag.Summary = desc.Summary
	protoDiag.Detail = desc.Detail

	srcRngs := diag.Source()
	if srcRngs.Subject != nil {
		protoDiag.Subject = sourceRangeToProto(*srcRngs.Subject)
	}
	if srcRngs.Context != nil {
		protoDiag.Context = sourceRangeToProto(*srcRngs.Context)
	}

	return protoDiag
}

func sourceRangeToProto(rng tfdiags.SourceRange) *terraform1.SourceRange {
	return &terraform1.SourceRange{
		// RPC API operations use source address syntax for "filename" by
		// convention, because the physical filesystem layout is an
		// implementation detail.
		SourceAddr: rng.Filename,

		Start: sourcePosToProto(rng.Start),
		End:   sourcePosToProto(rng.End),
	}
}

func sourceRangeFromProto(protoRng *terraform1.SourceRange) tfdiags.SourceRange {
	return tfdiags.SourceRange{
		Filename: protoRng.SourceAddr,
		Start:    sourcePosFromProto(protoRng.Start),
		End:      sourcePosFromProto(protoRng.End),
	}
}

func sourceRangeFromHCL(rng hcl.Range) tfdiags.SourceRange {
	return tfdiags.SourceRange{
		Filename: rng.Filename,
		Start: tfdiags.SourcePos{
			Line:   rng.Start.Line,
			Column: rng.Start.Column,
			Byte:   rng.Start.Byte,
		},
		End: tfdiags.SourcePos{
			Line:   rng.End.Line,
			Column: rng.End.Column,
			Byte:   rng.End.Byte,
		},
	}
}

func sourcePosToProto(pos tfdiags.SourcePos) *terraform1.SourcePos {
	return &terraform1.SourcePos{
		Byte:   int64(pos.Byte),
		Line:   int64(pos.Line),
		Column: int64(pos.Column),
	}
}

func sourcePosFromProto(protoPos *terraform1.SourcePos) tfdiags.SourcePos {
	return tfdiags.SourcePos{
		Byte:   int(protoPos.Byte),
		Line:   int(protoPos.Line),
		Column: int(protoPos.Column),
	}
}

func dynamicTypedValueFromProto(protoVal *stacks.DynamicValue) (cty.Value, error) {
	if len(protoVal.Msgpack) == 0 {
		return cty.DynamicVal, fmt.Errorf("uses unsupported serialization format (only MessagePack is supported)")
	}
	v, err := msgpack.Unmarshal(protoVal.Msgpack, cty.DynamicPseudoType)
	if err != nil {
		return cty.DynamicVal, fmt.Errorf("invalid serialization: %w", err)
	}
	// FIXME: Incredibly imprecise handling of sensitive values. We should
	// actually decode the attribute paths and mark individual leaf attributes
	// that are sensitive, but for now we'll just mark the whole thing as
	// sensitive if any part of it is sensitive.
	if len(protoVal.Sensitive) != 0 {
		v = v.Mark(marks.Sensitive)
	}
	return v, nil
}

func externalInputValuesFromProto(protoVals map[string]*stacks.DynamicValueWithSource) (map[stackaddrs.InputVariable]stackruntime.ExternalInputValue, error) {
	if len(protoVals) == 0 {
		return nil, nil
	}
	var err error
	ret := make(map[stackaddrs.InputVariable]stackruntime.ExternalInputValue, len(protoVals))
	for name, protoVal := range protoVals {
		v, moreErr := externalInputValueFromProto(protoVal)
		if moreErr != nil {
			err = errors.Join(err, fmt.Errorf("%s: %w", name, moreErr))
		}
		ret[stackaddrs.InputVariable{Name: name}] = v
	}
	return ret, err
}

func externalInputValueFromProto(protoVal *stacks.DynamicValueWithSource) (stackruntime.ExternalInputValue, error) {
	v, err := dynamicTypedValueFromProto(protoVal.Value)
	if err != nil {
		return stackruntime.ExternalInputValue{}, nil
	}
	rng := sourceRangeFromProto(protoVal.SourceRange)
	return stackruntime.ExternalInputValue{
		Value:    v,
		DefRange: rng,
	}, nil
}

func componentInstancePolicyEvaluationProto(componentAddr stackaddrs.AbsComponentInstance, policyResults *plans.PolicyResults) *stacks.PolicyEvaluationResponse {
	results := make([]*stacks.PolicyResult, 0)
	infos := make([]*stacks.PolicyInfo, 0)
	policyDiags := make([]*stacks.PolicyDiagnostic, 0)

	for addr, result := range policyResults.Iter() {
		results = append(results, policyResultsToProto(addr, result.EvaluationResponse.Policies)...)
		infos = append(infos, policyInfosToProto(addr, result.EvaluationResponse.Enforcements)...)
		policyDiags = append(policyDiags, policyDiagsToProto(addr, result.EvaluationResponse.Diagnostics)...)
	}

	return &stacks.PolicyEvaluationResponse{
		Addr: &stacks.ComponentInstanceInStackAddr{
			ComponentAddr:         stackaddrs.ConfigComponentForAbsInstance(componentAddr).String(),
			ComponentInstanceAddr: componentAddr.String(),
		},
		Results:     results,
		Infos:       infos,
		Diagnostics: policyDiags,
	}
}

func policyEvaluateResultToProto(result policy.EvaluateResult) stacks.EvaluateResult {
	switch result {
	case policy.InvalidResult:
		return stacks.EvaluateResult_INVALID_EVALUATE_RESULT
	case policy.UnknownResult:
		return stacks.EvaluateResult_UNKNOWN_EVALUATE_RESULT
	case policy.PolicyErrorResult:
		return stacks.EvaluateResult_ERROR_EVALUATE_RESULT
	case policy.AllowResult:
		return stacks.EvaluateResult_ALLOW_EVALUATE_RESULT
	case policy.DenyResult:
		return stacks.EvaluateResult_DENY_EVALUATE_RESULT
	case policy.SetupErrorResult:
		return stacks.EvaluateResult_SETUP_ERROR_EVALUATE_RESULT
	default:
		// should be exhaustive
		panic(fmt.Errorf("unhandled policy.EvaluateResult type: %T", result))
	}
}

func policyResultsToProto(addr string, policies []*policy.Policy) []*stacks.PolicyResult {
	protoPolicyResults := make([]*stacks.PolicyResult, len(policies))
	for i, policy := range policies {
		result := stacks.PolicyResult{
			TargetAddress:  addr,
			PolicyMetadata: policyMetadataToProto(policy, nil),
			Result:         policyEvaluateResultToProto(policy.Result),
		}
		protoPolicyResults[i] = &result
	}
	return protoPolicyResults
}

func policyMetadataToProto(policyObj *policy.Policy, enforceIndex *int32) *stacks.PolicyMetaData {
	if policyObj == nil {
		return nil
	}
	metadata := &stacks.PolicyMetaData{
		PolicySetName:    policyObj.PolicySetName,
		PolicyName:       policyObj.Address,
		FileName:         policyObj.Filename,
		EnforcementLevel: policyObj.EnforcementLevel,
	}

	if enforceIndex != nil {
		metadata.EnforceIndex = *enforceIndex
	}

	return metadata
}

func policyInfosToProto(addr string, enforcements []policy.EnforcementResult) []*stacks.PolicyInfo {
	protoPolicyInfos := make([]*stacks.PolicyInfo, 0)

	for _, enforcement := range enforcements {
		if enforcement.Message == "" {
			continue
		}
		protoPolicyMetadata := policyMetadataToProto(enforcement.Policy, &enforcement.BlockIndex)

		var protoPolicySnippet *stacks.PolicySnippet
		if snippet := enforcement.Snippet; snippet != nil {
			protoPolicySnippet = &stacks.PolicySnippet{
				Code:                 snippet.Code,
				StartLine:            snippet.StartLine,
				HighlightStartOffset: snippet.HighlightStartOffset,
				HighlightEndOffset:   snippet.HighlightEndOffset,
			}
			if snippet.Context != nil {
				protoPolicySnippet.Context = *snippet.Context
			}
		}

		var protoPolicyRange *terraform1.SourceRange
		if enforcement.Range != nil {
			rng := sourceRangeFromHCL(*enforcement.Range)
			protoPolicyRange = &terraform1.SourceRange{
				SourceAddr: enforcement.Range.Filename,
				Start:      sourcePosToProto(rng.Start),
				End:        sourcePosToProto(rng.End),
			}
		}

		protoPolicyInfos = append(protoPolicyInfos, &stacks.PolicyInfo{
			TargetAddress:  addr,
			Result:         policyEvaluateResultToProto(enforcement.Result),
			Message:        enforcement.Message,
			PolicySnippet:  protoPolicySnippet,
			PolicyMetadata: protoPolicyMetadata,
			PolicyRange:    protoPolicyRange,
		})
	}

	return protoPolicyInfos
}

func policyDiagsToProto(addr string, policyDiags policy.Diagnostics) []*stacks.PolicyDiagnostic {
	protoPolicyDiags := make([]*stacks.PolicyDiagnostic, len(policyDiags))

	for i, diag := range policyDiags {
		desc := diag.Description()
		extra := tfdiags.ExtraInfo[*policy.PolicyExtra](diag)

		policyDiag := stacks.PolicyDiagnostic{
			TargetAddress: addr,
			Diagnostic: &terraform1.Diagnostic{
				Severity: terraform1.Diagnostic_ERROR,
				Summary:  desc.Summary,
				Detail:   desc.Detail,
			},
		}

		if extra != nil {
			if extra.Severity == hcl.DiagWarning {
				policyDiag.Diagnostic.Severity = terraform1.Diagnostic_WARNING
			}

			policyDiag.Result = policyEvaluateResultToProto(extra.Result)
			policyDiag.PolicyMetadata = policyMetadataToProto(&extra.Policy, extra.EnforceIndex)

			if snippet := extra.Snippet; snippet != nil {
				policyDiag.PolicySnippet = &stacks.PolicySnippet{
					Code:                 snippet.Code,
					StartLine:            snippet.StartLine,
					HighlightStartOffset: snippet.HighlightStartOffset,
					HighlightEndOffset:   snippet.HighlightEndOffset,
				}
				if snippet.Context != nil {
					policyDiag.PolicySnippet.Context = *snippet.Context
				}
			}

			if rng := extra.Range; rng != nil && rng.Subject != nil {
				policyDiag.PolicyRange = &terraform1.SourceRange{
					SourceAddr: rng.Subject.Filename,
				}
				if start := rng.Subject.Start; start != nil {
					policyDiag.PolicyRange.Start = &terraform1.SourcePos{
						Line:   start.Line,
						Column: start.Column,
						Byte:   start.Byte,
					}
				}
				if end := rng.Subject.End; end != nil {
					policyDiag.PolicyRange.End = &terraform1.SourcePos{
						Line:   end.Line,
						Column: end.Column,
						Byte:   end.Byte,
					}
				}
			}

			policyDiag.ExpressionValues = policyExpressionValuesToProto(extra.ExpressionValues)
		}

		if src := diag.Source(); src.Subject != nil {
			policyDiag.Diagnostic.Subject = sourceRangeToProto(*src.Subject)
		}
		if src := diag.Source(); src.Context != nil {
			policyDiag.Diagnostic.Context = sourceRangeToProto(*src.Context)
		}

		protoPolicyDiags[i] = &policyDiag
	}

	return protoPolicyDiags
}

func policyExpressionValuesToProto(policyExpressionValues []*proto.ExpressionValue) []*stacks.ExpressionValue {
	if len(policyExpressionValues) == 0 {
		return nil
	}

	expressionValues := make([]*stacks.ExpressionValue, 0, len(policyExpressionValues))
	seen := make(map[string]struct{}, len(policyExpressionValues))

	for _, val := range policyExpressionValues {
		path, err := val.Traversal.ToCtyPath()
		if err != nil {
			continue // then we can't display this value
		}

		exprValue := &stacks.ExpressionValue{
			Traversal: stacks.NewAttributePath(path),
		}

		strPath := ctyPathStr(path)
		if _, exists := seen[strPath]; exists {
			continue
		}
		seen[strPath] = struct{}{}

		exprValue.Value = val.Value
		expressionValues = append(expressionValues, exprValue)
	}

	return expressionValues
}

func ctyPathStr(path cty.Path) string {
	// This is a specialized subset of traversal rendering tailored to
	// producing a string that can be used to detect duplicate cty paths.
	// It is not comprehensive nor intended to be used for other purposes.

	var buf bytes.Buffer
	first := true
	for _, step := range path {
		switch tStep := step.(type) {
		case cty.GetAttrStep:
			if !first {
				buf.WriteByte('.')
			}
			buf.WriteString(tStep.Name)
		case cty.IndexStep:
			buf.WriteByte('[')
			if keyTy := tStep.Key.Type(); keyTy.IsPrimitiveType() {
				buf.WriteString(tfdiags.CompactValueStr(tStep.Key))
			} else {
				// We'll just use a placeholder for more complex values,
				// since otherwise our result could grow ridiculously long.
				buf.WriteString("...")
			}
			buf.WriteByte(']')
		}
		first = false
	}
	return buf.String()
}
