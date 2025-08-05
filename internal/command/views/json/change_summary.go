// Copyright (c) HashiCorp, Inc.
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
	Operation        Operation `json:"operation"`
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
			buf.WriteString(fmt.Sprintf(" %d actions invoked.", cs.ActionInvocation))
		}
	case OperationDestroyed:
		buf.WriteString(fmt.Sprintf("Destroy complete! Resources: %d destroyed.", cs.Remove))
		if cs.ActionInvocation > 0 {
			buf.WriteString(fmt.Sprintf(" %d actions invoked.", cs.ActionInvocation))
		}
	case OperationPlanned:
		buf.WriteString("Plan: ")
		if cs.Import > 0 {
			buf.WriteString(fmt.Sprintf("%d to import, ", cs.Import))
		}
		buf.WriteString(fmt.Sprintf("%d to add, %d to change, %d to destroy.", cs.Add, cs.Change, cs.Remove))
		if cs.ActionInvocation > 0 {
			buf.WriteString(fmt.Sprintf(" %d actions to be invoked.", cs.ActionInvocation))
		}
	default:
		buf.WriteString(fmt.Sprintf("%s: %d add, %d change, %d destroy", cs.Operation, cs.Add, cs.Change, cs.Remove))
	}

	return buf.String()
}
