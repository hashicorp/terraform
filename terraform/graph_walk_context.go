package terraform

import (
	"context"
	"log"
	"sync"

	"github.com/zclconf/go-cty/cty"

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
	Context            *Context
	Operation          walkOperation
	StopContext        context.Context
	RootVariableValues InputValues

	// This is an output. Do not set this, nor read it while a graph walk
	// is in progress.
	NonFatalDiagnostics tfdiags.Diagnostics

	errorLock          sync.Mutex
	once               sync.Once
	contexts           map[string]*BuiltinEvalContext
	contextLock        sync.Mutex
	variableValues     map[string]map[string]cty.Value
	variableValuesLock sync.Mutex
	providerCache      map[string]ResourceProvider
	providerSchemas    map[string]*ProviderSchema
	providerLock       sync.Mutex
	provisionerCache   map[string]ResourceProvisioner
	provisionerSchemas map[string]*configschema.Block
	provisionerLock    sync.Mutex
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

	// Our evaluator shares some locks with the main context and the walker
	// so that we can safely run multiple evaluations at once across
	// different modules.
	evaluator := &Evaluator{
		Meta:               w.Context.meta,
		Config:             w.Context.config,
		Operation:          w.Operation,
		State:              w.Context.state,
		StateLock:          &w.Context.stateLock,
		Schemas:            w.Context.schemas,
		VariableValues:     w.variableValues,
		VariableValuesLock: &w.variableValuesLock,
	}

	ctx := &BuiltinEvalContext{
		StopContext:         w.StopContext,
		PathValue:           path,
		Hooks:               w.Context.hooks,
		InputValue:          w.Context.uiInput,
		Components:          w.Context.components,
		Schemas:             w.Context.schemas,
		ProviderCache:       w.providerCache,
		ProviderInputConfig: w.Context.providerInputConfig,
		ProviderLock:        &w.providerLock,
		ProvisionerCache:    w.provisionerCache,
		ProvisionerLock:     &w.provisionerLock,
		DiffValue:           w.Context.diff,
		DiffLock:            &w.Context.diffLock,
		StateValue:          w.Context.state,
		StateLock:           &w.Context.stateLock,
		Evaluator:           evaluator,
		VariableValues:      w.variableValues,
		VariableValuesLock:  &w.variableValuesLock,
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
		log.Printf("[WARN] %s: %s", dag.VertexName(v), nferr)
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
	w.variableValues = make(map[string]map[string]cty.Value)

	// Populate root module variable values. Other modules will be populated
	// during the graph walk.
	w.variableValues[""] = make(map[string]cty.Value)
	for k, iv := range w.RootVariableValues {
		w.variableValues[""][k] = iv.Value
	}
}
