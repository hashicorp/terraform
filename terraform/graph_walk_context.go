package terraform

import (
	"fmt"
	"sync"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/dag"
)

// ContextGraphWalker is the GraphWalker implementation used with the
// Context struct to walk and evaluate the graph.
type ContextGraphWalker struct {
	NullGraphWalker

	// Configurable values
	Context   *Context
	Operation walkOperation

	// Outputs, do not set these. Do not read these while the graph
	// is being walked.
	ValidationWarnings []string
	ValidationErrors   []error

	errorLock           sync.Mutex
	once                sync.Once
	contexts            map[string]*BuiltinEvalContext
	contextLock         sync.Mutex
	providerCache       map[string]ResourceProvider
	providerConfigCache map[string]*ResourceConfig
	providerLock        sync.Mutex
	provisionerCache    map[string]ResourceProvisioner
	provisionerLock     sync.Mutex
}

func (w *ContextGraphWalker) EnterGraph(g *Graph) EvalContext {
	w.once.Do(w.init)

	w.contextLock.Lock()
	defer w.contextLock.Unlock()

	// If we already have a context for this path cached, use that
	key := PathCacheKey(g.Path)
	if ctx, ok := w.contexts[key]; ok {
		return ctx
	}

	// Variables should be our context variables, but these only apply
	// to the root module. As we enter subgraphs, we don't want to set
	// variables, which is set by the SetVariables EvalContext function.
	variables := w.Context.variables
	if len(g.Path) > 1 {
		// We're in a submodule, the variables should be empty
		variables = make(map[string]string)
	}

	ctx := &BuiltinEvalContext{
		PathValue:           g.Path,
		Hooks:               w.Context.hooks,
		InputValue:          w.Context.uiInput,
		Providers:           w.Context.providers,
		ProviderCache:       w.providerCache,
		ProviderConfigCache: w.providerConfigCache,
		ProviderInputConfig: w.Context.providerInputConfig,
		ProviderLock:        &w.providerLock,
		Provisioners:        w.Context.provisioners,
		ProvisionerCache:    w.provisionerCache,
		ProvisionerLock:     &w.provisionerLock,
		DiffValue:           w.Context.diff,
		DiffLock:            &w.Context.diffLock,
		StateValue:          w.Context.state,
		StateLock:           &w.Context.stateLock,
		Interpolater: &Interpolater{
			Operation: w.Operation,
			Module:    w.Context.module,
			State:     w.Context.state,
			StateLock: &w.Context.stateLock,
			Variables: variables,
		},
	}

	w.contexts[key] = ctx
	return ctx
}

func (w *ContextGraphWalker) EnterEvalTree(v dag.Vertex, n EvalNode) EvalNode {
	// Acquire a lock on the semaphore
	w.Context.parallelSem.Acquire()

	// We want to filter the evaluation tree to only include operations
	// that belong in this operation.
	return EvalFilter(n, EvalNodeFilterOp(w.Operation))
}

func (w *ContextGraphWalker) ExitEvalTree(
	v dag.Vertex, output interface{}, err error) error {
	// Release the semaphore
	w.Context.parallelSem.Release()

	if err == nil {
		return nil
	}

	// Acquire the lock because anything is going to require a lock.
	w.errorLock.Lock()
	defer w.errorLock.Unlock()

	// Try to get a validation error out of it. If its not a validation
	// error, then just record the normal error.
	verr, ok := err.(*EvalValidateError)
	if !ok {
		return err
	}

	for _, msg := range verr.Warnings {
		w.ValidationWarnings = append(
			w.ValidationWarnings,
			fmt.Sprintf("%s: %s", dag.VertexName(v), msg))
	}
	for _, e := range verr.Errors {
		w.ValidationErrors = append(
			w.ValidationErrors,
			errwrap.Wrapf(fmt.Sprintf("%s: {{err}}", dag.VertexName(v)), e))
	}

	return nil
}

func (w *ContextGraphWalker) init() {
	w.contexts = make(map[string]*BuiltinEvalContext, 5)
	w.providerCache = make(map[string]ResourceProvider, 5)
	w.providerConfigCache = make(map[string]*ResourceConfig, 5)
	w.provisionerCache = make(map[string]ResourceProvisioner, 5)
}
