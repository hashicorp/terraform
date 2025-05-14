// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"errors"
	"fmt"

	"github.com/zclconf/go-cty/cty"
	msgpack "github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/terraform/internal/lang/marks"
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
