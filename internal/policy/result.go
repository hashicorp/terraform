// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import "github.com/hashicorp/terraform/internal/policy/proto"

type EvaluateResult int

//go:generate go tool golang.org/x/tools/cmd/stringer -type=EvaluateResult

const (
	InvalidResult EvaluateResult = iota
	UnknownResult
	PolicyErrorResult
	AllowResult
	DenyResult
	SetupErrorResult
)

func ResultFromProto(result proto.EvaluateResult) EvaluateResult {
	switch result {
	case proto.EvaluateResult_INVALID_EVALUATE_RESULT:
		return InvalidResult
	case proto.EvaluateResult_UNKNOWN_EVALUATE_RESULT:
		return UnknownResult
	case proto.EvaluateResult_ERROR_EVALUATE_RESULT:
		return PolicyErrorResult
	case proto.EvaluateResult_ALLOW_EVALUATE_RESULT:
		return AllowResult
	case proto.EvaluateResult_DENY_EVALUATE_RESULT:
		return DenyResult
	case proto.EvaluateResult_SETUP_ERROR_EVALUATE_RESULT:
		return SetupErrorResult
	default:
		// should be exhaustive
		panic("unhandled EvaluateResult")
	}
}
