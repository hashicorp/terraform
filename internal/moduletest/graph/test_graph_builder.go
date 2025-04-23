// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestGraphBuilder is a GraphBuilder implementation that builds a graph for
// a terraform test file. The file may contain multiple runs, and each run may have
// dependencies on other runs.
type TestGraphBuilder struct {
	File           *moduletest.File
	GlobalVars     map[string]backendrun.UnparsedVariableValue
	ContextOpts    *terraform.ContextOpts
	BackendFactory func(string) backend.InitFn
	StateManifest  *TestManifest
	CommandMode    moduletest.CommandMode
}

type graphOptions struct {
	File          *moduletest.File
	GlobalVars    map[string]backendrun.UnparsedVariableValue
	ContextOpts   *terraform.ContextOpts
	StateManifest *TestManifest
	CommandMode   moduletest.CommandMode
	EvalContext   *EvalContext
}

// See GraphBuilder
func (b *TestGraphBuilder) Build(ctx *EvalContext) (*terraform.Graph, tfdiags.Diagnostics) {
	log.Printf("[TRACE] building graph for terraform test")
	opts := &graphOptions{
		File:          b.File,
		GlobalVars:    b.GlobalVars,
		ContextOpts:   b.ContextOpts,
		StateManifest: b.StateManifest,
		CommandMode:   b.CommandMode,
		EvalContext:   ctx,
	}
	return (&terraform.BasicGraphBuilder{
		Steps: b.Steps(opts),
		Name:  "TestGraphBuilder",
	}).Build(addrs.RootModuleInstance)
}

// See GraphBuilder
func (b *TestGraphBuilder) Steps(opts *graphOptions) []terraform.GraphTransformer {
	steps := []terraform.GraphTransformer{
		&TestRunTransformer{opts},
		&TestStateTransformer{graphOptions: opts, BackendFactory: b.BackendFactory},
		terraform.DynamicTransformer(validateRunConfigs),
		&TestProvidersTransformer{},
		terraform.DynamicTransformer(func(g *terraform.Graph) error {
			cleanup := &CleanupSubGraph{opts: opts}
			g.Add(cleanup)

			for v := range dag.ExcludeSeq(g.VerticesSeq(), func(*CleanupSubGraph) {}) {
				if g.UpEdges(v).Len() == 0 {
					g.Connect(dag.BasicEdge(cleanup, v))
				}
			}

			return nil
		}),
		&CloseTestGraphTransformer{},
		&terraform.TransitiveReductionTransformer{},
	}

	return steps
}

func validateRunConfigs(g *terraform.Graph) error {
	for node := range dag.SelectSeq(g.VerticesSeq(), runFilter) {
		diags := node.run.Config.Validate(node.run.ModuleConfig)
		node.run.Diagnostics = node.run.Diagnostics.Append(diags)
		if diags.HasErrors() {
			node.run.Status = moduletest.Error
		}
	}
	return nil
}

// dynamicNode is a helper node which can be added to the graph to execute
// a dynamic function at some desired point in the graph.
type dynamicNode struct {
	eval func(*EvalContext) tfdiags.Diagnostics
}

func (n *dynamicNode) Execute(evalCtx *EvalContext) tfdiags.Diagnostics {
	return n.eval(evalCtx)
}

func Walk(g *terraform.Graph, ctx *EvalContext) tfdiags.Diagnostics {

	// Walk the graph.
	walkFn := func(v dag.Vertex) (diags tfdiags.Diagnostics) {
		if ctx.Cancelled() {
			// If the graph walk has been cancelled, the node should just return immediately.
			// For now, this means a hard stop has been requested, in this case we don't
			// even stop to mark future test runs as having been skipped. They'll
			// just show up as pending in the printed summary. We will quickly
			// just mark the overall file status has having errored to indicate
			// it was interrupted.
			return
		}

		// the walkFn is called asynchronously, and needs to be recovered
		// separately in the case of a panic.
		defer logging.PanicHandler()

		log.Printf("[TRACE] vertex %q: starting visit (%T)", dag.VertexName(v), v)

		defer func() {
			if r := recover(); r != nil {
				// If the walkFn panics, we get confusing logs about how the
				// visit was complete. To stop this, we'll catch the panic log
				// that the vertex panicked without finishing and re-panic.
				log.Printf("[ERROR] vertex %q panicked", dag.VertexName(v))
				panic(r) // re-panic
			}

			if diags.HasErrors() {
				for _, diag := range diags {
					if diag.Severity() == tfdiags.Error {
						desc := diag.Description()
						log.Printf("[ERROR] vertex %q error: %s", dag.VertexName(v), desc.Summary)
					}
				}
				log.Printf("[TRACE] vertex %q: visit complete, with errors", dag.VertexName(v))
			} else {
				log.Printf("[TRACE] vertex %q: visit complete", dag.VertexName(v))
			}
		}()

		// Acquire a lock on the semaphore
		ctx.semaphore.Acquire()
		defer ctx.semaphore.Release()

		if executable, ok := v.(GraphNodeExecutable); ok {
			diags = executable.Execute(ctx)
		}
		return
	}

	return g.AcyclicGraph.Walk(walkFn)
}
