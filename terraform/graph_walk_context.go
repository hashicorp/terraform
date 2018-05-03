package terraform

import (
	"context"
	"log"
	"sync"

	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/dag"
)

// ContextGraphWalker is the GraphWalker implementation used with the
// Context struct to walk and evaluate the graph.
type ContextGraphWalker struct {
	NullGraphWalker

	// Configurable values
	Context     *Context
	Operation   walkOperation
	StopContext context.Context

	// This is an output. Do not set this, nor read it while a graph walk
	// is in progress.
	NonFatalDiagnostics tfdiags.Diagnostics

	errorLock           sync.Mutex
	once                sync.Once
	contexts            map[string]*BuiltinEvalContext
	contextLock         sync.Mutex
	interpolaterVars    map[string]map[string]interface{}
	interpolaterVarLock sync.Mutex
	providerCache       map[string]ResourceProvider
	providerSchemas     map[string]*ProviderSchema
	providerLock        sync.Mutex
	provisionerCache    map[string]ResourceProvisioner
	provisionerSchemas  map[string]*configschema.Block
	provisionerLock     sync.Mutex
}

func (w *ContextGraphWalker) EnterPath(path addrs.ModuleInstance) EvalContext {
	w.once.Do(w.init)

	w.contextLock.Lock()
	defer w.contextLock.Unlock()

	// If we already have a context for this path cached, use that
	key := path.String()
	if ctx, ok := w.contexts[key]; ok {
		return ctx
	}

	// Setup the variables for this interpolater
	variables := make(map[string]interface{})
	if len(path) <= 1 {
		for k, v := range w.Context.variables {
			variables[k] = v
		}
	}
	w.interpolaterVarLock.Lock()
	if m, ok := w.interpolaterVars[key]; ok {
		for k, v := range m {
			variables[k] = v
		}
	}
	w.interpolaterVars[key] = variables
	w.interpolaterVarLock.Unlock()

	// Our evaluator shares some locks with the main context and the walker
	// so that we can safely run multiple evaluations at once across
	// different modules.
	evaluator := &Evaluator{
		Meta:            w.Context.meta,
		Config:          w.Context.config,
		State:           w.Context.state,
		StateLock:       &w.Context.stateLock,
		ProviderSchemas: w.providerSchemas,
		ProvidersLock:   &w.providerLock,

		// FIXME: This was a design mistake on the evaluator, which should
		// get replaced with something like the interpolatorVars thing above
		// once we verify exactly how that was used in the old Interpolator
		// codepath.
		RootVariableValues: map[string]*InputValue{},
	}

	ctx := &BuiltinEvalContext{
		StopContext:         w.StopContext,
		PathValue:           path,
		Hooks:               w.Context.hooks,
		InputValue:          w.Context.uiInput,
		Components:          w.Context.components,
		ProviderCache:       w.providerCache,
		ProviderInputConfig: w.Context.providerInputConfig,
		ProviderSchemas:     w.providerSchemas,
		ProviderLock:        &w.providerLock,
		ProvisionerCache:    w.provisionerCache,
		ProvisionerLock:     &w.provisionerLock,
		DiffValue:           w.Context.diff,
		DiffLock:            &w.Context.diffLock,
		StateValue:          w.Context.state,
		StateLock:           &w.Context.stateLock,
		Evaluator:           evaluator,
	}

	w.contexts[key] = ctx
	return ctx
}

func (w *ContextGraphWalker) EnterEvalTree(v dag.Vertex, n EvalNode) EvalNode {
	log.Printf("[TRACE] [%s] Entering eval tree: %s", w.Operation, dag.VertexName(v))

	// Acquire a lock on the semaphore
	w.Context.parallelSem.Acquire()

	// We want to filter the evaluation tree to only include operations
	// that belong in this operation.
	return EvalFilter(n, EvalNodeFilterOp(w.Operation))
}

func (w *ContextGraphWalker) ExitEvalTree(v dag.Vertex, output interface{}, err error) tfdiags.Diagnostics {
	log.Printf("[TRACE] [%s] Exiting eval tree: %s", w.Operation, dag.VertexName(v))

	// Release the semaphore
	w.Context.parallelSem.Release()

	if err == nil {
		return nil
	}

	// Acquire the lock because anything is going to require a lock.
	w.errorLock.Lock()
	defer w.errorLock.Unlock()

	// If the error is non-fatal then we'll accumulate its diagnostics in our
	// non-fatal list, rather than returning it directly, so that the graph
	// walk can continue.
	if nferr, ok := err.(tfdiags.NonFatalError); ok {
		w.NonFatalDiagnostics = w.NonFatalDiagnostics.Append(nferr.Diagnostics)
		return nil
	}

	// Otherwise, we'll let our usual diagnostics machinery figure out how to
	// unpack this as one or more diagnostic messages and return that. If we
	// get down here then the returned diagnostics will contain at least one
	// error, causing the graph walk to halt.
	var diags tfdiags.Diagnostics
	diags = diags.Append(err)
	return diags
}

func (w *ContextGraphWalker) init() {
	w.contexts = make(map[string]*BuiltinEvalContext)
	w.providerCache = make(map[string]ResourceProvider)
	w.providerSchemas = make(map[string]*ProviderSchema)
	w.provisionerCache = make(map[string]ResourceProvisioner)
	w.provisionerSchemas = make(map[string]*configschema.Block)
	w.interpolaterVars = make(map[string]map[string]interface{})
}
