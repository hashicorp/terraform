// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package proto

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

var (
	_ tfdiags.DiagnosticExtraUnwrapper = (*diagnosticExtra)(nil)
)

type diagnosticExtra struct {
	next interface{}
}

func (d *diagnosticExtra) UnwrapDiagnosticExtra() interface{} {
	return d.next
}

// SnippetExtra is an extra containing a code snippet. As source information
// is lost when the diagnostic is translated to a protocol buffer, this extra
// captures the relevant parts of the source code.
type SnippetExtra struct {
	diagnosticExtra
	Snippet *Snippet
}

// RangeExtra is an extra containing the file information about the policy
// that produced this diagnostic.
type RangeExtra struct {
	diagnosticExtra
	Subject *Range
	Context *Range
}

// ExpressionValuesExtra is an extra containing expression values. As HCL
// evaluation contexts are lost when the diagnostic is translated to a protocol
// buffer, this extra captures the expression values from the context.
type ExpressionValuesExtra struct {
	diagnosticExtra
	ExpressionValues []*ExpressionValue
}

// FunctionCallExtra is an extra containing a function call. As HCL evaluation
// contexts are lost when the diagnostic is translated to a protocol buffer,
// this extra captures the function call from the context.
type FunctionCallExtra struct {
	diagnosticExtra
	FunctionCall string
}

// EvaluateResultExtra is an extra containing the evaluate result that this
// particular diagnostic would cause.
type EvaluateResultExtra struct {
	diagnosticExtra
	EvaluateResult EvaluateResult
}

// PolicyExtra simply marks a diagnostic as having been produced by the policy
// engine.
type PolicyExtra struct {
	diagnosticExtra

	PolicySet PolicySetMeta
}

type PolicySetMeta struct {
	diagnosticExtra

	Name string
	Path string
}

type AttributeExtra struct {
	diagnosticExtra
	Attribute cty.Path
}
