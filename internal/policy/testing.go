// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"
)

func NewTestMockClient(t *testing.T) *MockClient {

	ret := &MockClient{}

	ret.EvaluateProviderResponse = &EvaluationResponse{Overall: AllowResult}
	ret.EvaluateResponse = &EvaluationResponse{Overall: AllowResult}
	ret.EvaluateModuleResponse = &EvaluationResponse{Overall: AllowResult}
	return ret
}
