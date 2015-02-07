package terraform

import (
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/dag"
)

// ContextGraphWalker is the GraphWalker implementation used with the
// Context struct to walk and evaluate the graph.
type ContextGraphWalker struct {
	NullGraphWalker

	Context   *Context2
	Operation walkOperation

	ErrorLock          sync.Mutex
	EvalError          error
	ValidationWarnings []string
	ValidationErrors   []error
}

func (w *ContextGraphWalker) EnterGraph(g *Graph) EvalContext {
	return &BuiltinEvalContext{
		Path:      g.Path,
		Providers: w.Context.providers,
		Interpolater: &Interpolater{
			Operation: w.Operation,
			Module:    w.Context.module,
			State:     w.Context.state,
			StateLock: &w.Context.stateLock,
			Variables: nil,
		},
	}
}

func (w *ContextGraphWalker) ExitEvalTree(
	v dag.Vertex, output interface{}, err error) {
	if err == nil {
		return
	}

	// Acquire the lock because anything is going to require a lock.
	w.ErrorLock.Lock()
	defer w.ErrorLock.Unlock()

	// Try to get a validation error out of it. If its not a validation
	// error, then just record the normal error.
	verr, ok := err.(*EvalValidateError)
	if !ok {
		// Some other error, record it
		w.EvalError = multierror.Append(w.EvalError, err)
		return
	}

	// Record the validation error
	w.ValidationWarnings = append(w.ValidationWarnings, verr.Warnings...)
	w.ValidationErrors = append(w.ValidationErrors, verr.Errors...)
}
