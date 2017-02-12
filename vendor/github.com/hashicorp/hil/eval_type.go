package hil

//go:generate stringer -type=EvalType eval_type.go

// EvalType represents the type of the output returned from a HIL
// evaluation.
type EvalType uint32

const (
	TypeInvalid EvalType = 0
	TypeString  EvalType = 1 << iota
	TypeBool
	TypeList
	TypeMap
	TypeUnknown
)
