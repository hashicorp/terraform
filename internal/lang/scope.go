package lang

import (
	"sync"

	"github.com/zclconf/go-cty/cty/function"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/experiments"
)

// Scope is the main type in this package, allowing dynamic evaluation of
// blocks and expressions based on some contextual information that informs
// which variables and functions will be available.
type Scope struct {
	// Data is used to resolve references in expressions.
	Data Data

	// SelfAddr is the address that the "self" object should be an alias of,
	// or nil if the "self" object should not be available at all.
	SelfAddr addrs.Referenceable

	// BaseDir is the base directory used by any interpolation functions that
	// accept filesystem paths as arguments.
	BaseDir string

	// PureOnly can be set to true to request that any non-pure functions
	// produce unknown value results rather than actually executing. This is
	// important during a plan phase to avoid generating results that could
	// then differ during apply.
	PureOnly bool

	funcs     map[string]function.Function
	funcsLock sync.Mutex

	// activeExperiments is an optional set of experiments that should be
	// considered as active in the module that this scope will be used for.
	// Callers can populate it by calling the SetActiveExperiments method.
	activeExperiments experiments.Set

	// ConsoleMode can be set to true to request any console-only functions are
	// included in this scope.
	ConsoleMode bool
}

// SetActiveExperiments allows a caller to declare that a set of experiments
// is active for the module that the receiving Scope belongs to, which might
// then cause the scope to activate some additional experimental behaviors.
func (s *Scope) SetActiveExperiments(active experiments.Set) {
	s.activeExperiments = active
}
