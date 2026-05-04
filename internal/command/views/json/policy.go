// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"github.com/hashicorp/terraform/internal/policy"
)

// PolicyInfo is like an info diagnostic from the policy engine,
// and as such borrows diagnostic-related structs to
// host source information such as range and snippet.
type PolicyInfo struct {
	Message       string             `json:"message,omitempty"`
	PolicyRange   *DiagnosticRange   `json:"policy_range,omitempty"`
	PolicySnippet *DiagnosticSnippet `json:"policy_snippet,omitempty"`

	// Range and Snippet are the terraform source information
	Range   *DiagnosticRange   `json:"range,omitempty"`
	Snippet *DiagnosticSnippet `json:"snippet,omitempty"`
}

type PolicyMetadata struct {
	PolicySetName    string `json:"policy_set_name,omitempty"`
	PolicySetPath    string `json:"policy_set_path,omitempty"`
	PolicyName       string `json:"policy_name,omitempty"`
	FileName         string `json:"file_name,omitempty"`
	EnforcementLevel string `json:"enforcement_level,omitempty"`
	EnforceIndex     *int32 `json:"enforce_index,omitempty"`
}

type EnforceMetadata struct {
	BlockIndex *int32 `json:"block_index,omitempty"`
}

func NewPolicyInfo(sourceCode []byte, enforcement policy.EnforcementResult) PolicyInfo {
	ret := PolicyInfo{
		Message: enforcement.Message,
	}

	if rng := enforcement.Range; rng != nil {
		ret.PolicyRange = &DiagnosticRange{
			Filename: rng.Filename,
			Start:    Pos(rng.Start),
			End:      Pos(rng.End),
		}
	}

	if snippet := enforcement.Snippet; snippet != nil {
		ret.PolicySnippet = &DiagnosticSnippet{
			Code:                 snippet.Code,
			StartLine:            int(snippet.StartLine),
			HighlightStartOffset: int(snippet.HighlightStartOffset),
			HighlightEndOffset:   int(snippet.HighlightEndOffset),
		}
		if snippet.Context != nil && snippet.Context.Context != "" {
			ret.PolicySnippet.Context = &snippet.Context.Context
		}
	}

	if rng := enforcement.LocalRange; rng != nil {
		ret.Range = &DiagnosticRange{
			Filename: rng.Filename,
			Start:    Pos(rng.Start),
			End:      Pos(rng.End),
		}
		if sourceCode != nil {
			ret.Snippet = snippetFromRange(sourceCode, *rng, *rng)
		}
	}

	return ret
}

func MetadataFromPolicy(policy policy.Policy) PolicyMetadata {
	return PolicyMetadata{
		PolicySetName:    policy.PolicySetName,
		PolicySetPath:    policy.Directory,
		PolicyName:       policy.Address,
		FileName:         policy.Filename,
		EnforcementLevel: policy.EnforcementLevel,
	}
}

func MetadataFromEnforcement(enforcement policy.EnforcementResult) PolicyMetadata {
	ret := MetadataFromPolicy(*enforcement.Policy)
	ret.EnforceIndex = &enforcement.BlockIndex
	return ret
}
