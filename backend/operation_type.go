package backend

//go:generate go run golang.org/x/tools/cmd/stringer -type=OperationType operation_type.go

// OperationType is an enum used with Operation to specify the operation
// type to perform for Terraform.
type OperationType uint

const (
	OperationTypeInvalid OperationType = iota
	OperationTypeRefresh
	OperationTypePlan
	OperationTypeApply
)
