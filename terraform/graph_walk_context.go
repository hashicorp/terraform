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

	// Configurable values
	Context   *Context2
	Operation walkOperation

	// Outputs, do not set these. Do not read these while the graph
	// is being walked.
	EvalError          error
	ValidationWarnings []string
	ValidationErrors   []error

	errorLock           sync.Mutex
	once                sync.Once
	providerCache       map[string]ResourceProvider
	providerConfigCache map[string]*ResourceConfig
	providerLock        sync.Mutex
	provisionerCache    map[string]ResourceProvisioner
	provisionerLock     sync.Mutex
}

func (w *ContextGraphWalker) EnterGraph(g *Graph) EvalContext {
	w.once.Do(w.init)

	return &BuiltinEvalContext{
		PathValue:           g.Path,
		Providers:           w.Context.providers,
		ProviderCache:       w.providerCache,
		ProviderConfigCache: w.providerConfigCache,
		ProviderLock:        &w.providerLock,
		Provisioners:        w.Context.provisioners,
		ProvisionerCache:    w.provisionerCache,
		ProvisionerLock:     &w.provisionerLock,
		Interpolater: &Interpolater{
			Operation: w.Operation,
			Module:    w.Context.module,
			State:     w.Context.state,
			StateLock: &w.Context.stateLock,
			Variables: w.Context.variables,
		},
	}
}

func (w *ContextGraphWalker) EnterEvalTree(v dag.Vertex, n EvalNode) EvalNode {
	// We want to filter the evaluation tree to only include operations
	// that belong in this operation.
	return EvalFilter(n, EvalNodeFilterOp(w.Operation))
}

func (w *ContextGraphWalker) ExitEvalTree(
	v dag.Vertex, output interface{}, err error) {
	if err == nil {
		return
	}

	// Acquire the lock because anything is going to require a lock.
	w.errorLock.Lock()
	defer w.errorLock.Unlock()

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

func (w *ContextGraphWalker) init() {
	w.providerCache = make(map[string]ResourceProvider, 5)
	w.providerConfigCache = make(map[string]*ResourceConfig, 5)
	w.provisionerCache = make(map[string]ResourceProvisioner, 5)
}
