// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import "testing"

func TestEvaluationResponseMerge(t *testing.T) {
	plain := EvaluationResponse{Overall: AllowResult, Enforcements: []EnforcementResult{{Result: AllowResult, Message: "plain"}}}
	related := EvaluationResponse{Overall: DenyResult, Enforcements: []EnforcementResult{{Result: DenyResult, Message: "related"}}}

	got := plain.Merge(related)
	if got.Overall != DenyResult || len(got.Enforcements) != 2 || got.Enforcements[1].Message != "related" {
		t.Errorf("wrong merge: %#v", got)
	}
	if got := related.Merge(plain); got.Overall != DenyResult {
		t.Errorf("merge lowered the result: %s", got.Overall)
	}
	if got := (EvaluationResponse{}).Merge(plain); got.Overall != AllowResult || len(got.Enforcements) != 1 {
		t.Errorf("merge into the zero response: %#v", got)
	}
	if got := plain.Merge(EvaluationResponse{Overall: AllowResult}); len(got.Enforcements) != 1 {
		t.Errorf("merge of an empty response: %#v", got)
	}
	if len(plain.Enforcements) != 1 {
		t.Errorf("merge changed its receiver")
	}
}

func TestEvaluationResponseMerge_zero(t *testing.T) {
	if got := (EvaluationResponse{}).Merge(EvaluationResponse{Overall: AllowResult}); got.Overall != AllowResult {
		t.Errorf("merge into the zero response: %s", got.Overall)
	}
}
