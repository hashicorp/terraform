package terraform

//go:generate go run golang.org/x/tools/cmd/stringer -type=walkOperation graph_walk_operation.go

// walkOperation is an enum which tells the walkContext what to do.
type walkOperation byte

const (
	walkInvalid walkOperation = iota
	walkApply
	walkPlan
	walkPlanDestroy
	walkRefresh
	walkValidate
	walkDestroy
	walkImport
	walkEval // used just to prepare EvalContext for expression evaluation, with no other actions
)
