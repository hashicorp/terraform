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

	// TODO(liamcervante): For now, we only include the import count in the plan
	//   output. This is because counting the imports during the apply is tricky
	//   and we need to use the hooks to get this information back. Currently,
	//   there is a PostImportState function on the hooks. This is almost
	//   certainly not being called in the right place for plannable import
	//   (since this hasn't been implemented yet).
	//
	//   We should absolutely fix this before we launch to alpha, but we can't
	//   do it right now. So we have implemented as much as we can (the plan)
	//   and will revisit this alongside the concrete implementation of the
	//   Terraform graph when we can concretely place the hook in the right
	//   place, introduce a new one, or modify an existing one.

	switch cs.Operation {
	case OperationApplied:
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
