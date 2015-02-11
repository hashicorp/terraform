package terraform

// This separate file for EvalType exists so that stringer below
// can work without error. See this thread for more details:
//
// http://comments.gmane.org/gmane.comp.lang.go.general/148740

//go:generate stringer -type=EvalType eval_type.go

// EvalType is the type of any value returned by an EvalNode. This is
// used for type checking.
type EvalType uint32

const (
	EvalTypeInvalid             EvalType = 0
	EvalTypeNull                EvalType = 1 << iota // nil
	EvalTypeConfig                                   // *ResourceConfig
	EvalTypeResourceProvider                         // ResourceProvider
	EvalTypeResourceProvisioner                      // ResourceProvisioner
	EvalTypeInstanceDiff                             // *InstanceDiff
	EvalTypeInstanceState                            // *InstanceState
)
