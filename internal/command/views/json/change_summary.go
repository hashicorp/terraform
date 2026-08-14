// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"fmt"
	"strings"
)

type Operation string

const (
	OperationApplied   Operation = "apply"
	OperationDestroyed Operation = "destroy"
	OperationPlanned   Operation = "plan"
)

type ChangeSummary struct {
	Add              int       `json:"add"`
	Change           int       `json:"change"`
	Import           int       `json:"import"`
	Remove           int       `json:"remove"`
	ActionInvocation int       `json:"action_invocation"`
	ActionFail       int       `json:"action_fail"`
	Operation        Operation `json:"operation"`

	// TODO:@austinvalle: Should we just emit a new log for this? (plan.Complete)
	// TODO:@austinvalle: Is it okay that this is re-used for apply? (where this won't be set)
	PartialPlan bool `json:"partial_plan,omitempty"`
}

// The summary strings for apply and plan are accidentally a public interface
// used by HCP Terraform and Terraform Enterprise, so the exact formats of
// these strings are important.
func (cs *ChangeSummary) String() string {
	var buf strings.Builder
	switch cs.Operation {
	case OperationApplied:
		buf.WriteString("Apply complete! Resources: ")
		if cs.Import > 0 {
			buf.WriteString(fmt.Sprintf("%d imported, ", cs.Import))
		}
		buf.WriteString(fmt.Sprintf("%d added, %d changed, %d destroyed.", cs.Add, cs.Change, cs.Remove))
		if cs.ActionInvocation > 0 {
			msg := fmt.Sprintf(" Actions: %d invoked", cs.ActionInvocation)
			if cs.ActionFail > 0 {
				msg = fmt.Sprintf("%s, %d failed", msg, cs.ActionFail)
			}

			buf.WriteString(msg + ".")
		}
	case OperationDestroyed:
		buf.WriteString(fmt.Sprintf("Destroy complete! Resources: %d destroyed.", cs.Remove))
		if cs.ActionInvocation > 0 {
			buf.WriteString(fmt.Sprintf(" Actions: %d invoked.", cs.ActionInvocation))
		}
	case OperationPlanned:
		buf.WriteString("Plan: ")
		if cs.Import > 0 {
			buf.WriteString(fmt.Sprintf("%d to import, ", cs.Import))
		}
		buf.WriteString(fmt.Sprintf("%d to add, %d to change, %d to destroy.", cs.Add, cs.Change, cs.Remove))
		if cs.ActionInvocation > 0 {
			buf.WriteString(fmt.Sprintf(" Actions: %d to invoke.", cs.ActionInvocation))
		}
	default:
		buf.WriteString(fmt.Sprintf("%s: %d add, %d change, %d destroy", cs.Operation, cs.Add, cs.Change, cs.Remove))
	}

	// TODO:@austinvalle: Should we also render if the plan is partial? The comment indicates this
	// is relied on by HCPT already, so don't want to break anything...

	return buf.String()
}
