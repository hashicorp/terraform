package terraform

//go:generate stringer -type=walkOperation graph_walk_operation.go

// walkOperation is an enum which tells the walkContext what to do.
type walkOperation byte

const (
	walkInvalid walkOperation = iota
	walkInput
	walkApply
	walkPlan
	walkPlanDestroy
	walkRefresh
	walkValidate
	walkDestroy
	walkImport
	walkEval // used just to prepare EvalContext for expression evaluation, with no other actions
)
