// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package json

import "fmt"

type Operation string

const (
	OperationApplied   Operation = "apply"
	OperationDestroyed Operation = "destroy"
	OperationPlanned   Operation = "plan"
)

type ChangeSummary struct {
	Add       int       `json:"add"`
	Change    int       `json:"change"`
	Import    int       `json:"import"`
	Remove    int       `json:"remove"`
	Operation Operation `json:"operation"`
}

// The summary strings for apply and plan are accidentally a public interface
// used by Terraform Cloud and Terraform Enterprise, so the exact formats of
// these strings are important.
func (cs *ChangeSummary) String() string {
	switch cs.Operation {
	case OperationApplied:
		if cs.Import > 0 {
			return fmt.Sprintf("Apply complete! Resources: %d imported, %d added, %d changed, %d destroyed.", cs.Import, cs.Add, cs.Change, cs.Remove)
		}
		return fmt.Sprintf("Apply complete! Resources: %d added, %d changed, %d destroyed.", cs.Add, cs.Change, cs.Remove)
	case OperationDestroyed:
		return fmt.Sprintf("Destroy complete! Resources: %d destroyed.", cs.Remove)
	case OperationPlanned:
		if cs.Import > 0 {
			return fmt.Sprintf("Plan: %d to import, %d to add, %d to change, %d to destroy.", cs.Import, cs.Add, cs.Change, cs.Remove)
		}
		return fmt.Sprintf("Plan: %d to add, %d to change, %d to destroy.", cs.Add, cs.Change, cs.Remove)
	default:
		return fmt.Sprintf("%s: %d add, %d change, %d destroy", cs.Operation, cs.Add, cs.Change, cs.Remove)
	}
}
