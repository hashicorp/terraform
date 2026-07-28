// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/policy"
)

// PolicyEvaluation holds the result of a policy evaluation for a single resource, module, or provider.
type PolicyEvaluation struct {
	EvaluationResponse policy.EvaluationResponse
	ConfigDeclRange    hcl.Range
}
