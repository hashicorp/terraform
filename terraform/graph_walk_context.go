package terraform

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/errwrap"
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

	// Outputs, do not set these. Do not read these while the graph
	// is being walked.
	ValidationWarnings []string
	ValidationErrors   []error

	errorLock           sync.Mutex
	once                sync.Once
	contexts            map[string]*BuiltinEvalContext
	contextLock         sync.Mutex
	interpolaterVars    map[string]map[string]interface{}
	interpolaterVarLock sync.Mutex
	providerCache       map[string]ResourceProvider
	providerConfigCache map[string]*ResourceConfig
	providerLock        sync.Mutex
	provisionerCache    map[string]ResourceProvisioner
	provisionerLock     sync.Mutex
}

func (w *ContextGraphWalker) EnterPath(path []string) EvalContext {
	w.once.Do(w.init)

	w.contextLock.Lock()
	defer w.contextLock.Unlock()

	// If we already have a context for this path cached, use that
	key := PathCacheKey(path)
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

	ctx := &BuiltinEvalContext{
		StopContext:         w.StopContext,
		PathValue:           path,
		Hooks:               w.Context.hooks,
		InputValue:          w.Context.uiInput,
		Components:          w.Context.components,
		ProviderCache:       w.providerCache,
		ProviderConfigCache: w.providerConfigCache,
		ProviderInputConfig: w.Context.providerInputConfig,
		ProviderLock:        &w.providerLock,
		ProvisionerCache:    w.provisionerCache,
		ProvisionerLock:     &w.provisionerLock,
		DiffValue:           w.Context.diff,
		DiffLock:            &w.Context.diffLock,
		StateValue:          w.Context.state,
		StateLock:           &w.Context.stateLock,
		Interpolater: &Interpolater{
			Operation:          w.Operation,
			Meta:               w.Context.meta,
			Module:             w.Context.module,
			State:              w.Context.state,
			StateLock:          &w.Context.stateLock,
			VariableValues:     variables,
			VariableValuesLock: &w.interpolaterVarLock,
		},
		InterpolaterVars:    w.interpolaterVars,
		InterpolaterVarLock: &w.interpolaterVarLock,
	}

	w.contexts[key] = ctx
	return ctx
}

func (w *ContextGraphWalker) EnterEvalTree(v dag.Vertex, n EvalNode) EvalNode {
	log.Printf("[TRACE] [%s] Entering eval tree: %s",
		w.Operation, dag.VertexName(v))

	// Acquire a lock on the semaphore
	w.Context.parallelSem.Acquire()

	// We want to filter the evaluation tree to only include operations
	// that belong in this operation.
	return EvalFilter(n, EvalNodeFilterOp(w.Operation))
}

func (w *ContextGraphWalker) ExitEvalTree(
	v dag.Vertex, output interface{}, err error) error {
	log.Printf("[TRACE] [%s] Exiting eval tree: %s",
		w.Operation, dag.VertexName(v))

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
	w.interpolaterVars = make(map[string]map[string]interface{}, 5)
}
