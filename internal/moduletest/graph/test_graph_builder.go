// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type GraphNodeExecutable interface {
	Execute(ctx *EvalContext)
}

// TestGraphBuilder is a GraphBuilder implementation that builds a graph for
// a terraform test file. The file may contain multiple runs, and each run may have
// dependencies on other runs.
type TestGraphBuilder struct {
	Config      *configs.Config
	File        *moduletest.File
	ContextOpts *terraform.ContextOpts
	CommandMode moduletest.CommandMode
}

type graphOptions struct {
	File        *moduletest.File
	ContextOpts *terraform.ContextOpts
}

// See GraphBuilder
func (b *TestGraphBuilder) Build() (*terraform.Graph, tfdiags.Diagnostics) {
	log.Printf("[TRACE] building graph for terraform test")
	return (&terraform.BasicGraphBuilder{
		Steps: b.Steps(),
		Name:  "TestGraphBuilder",
	}).Build(addrs.RootModuleInstance)
}

// See GraphBuilder
func (b *TestGraphBuilder) Steps() []terraform.GraphTransformer {
	opts := &graphOptions{
		File:        b.File,
		ContextOpts: b.ContextOpts,
	}
	steps := []terraform.GraphTransformer{
		&TestRunTransformer{opts: opts, mode: b.CommandMode},
		&TestVariablesTransformer{File: b.File},
		terraform.DynamicTransformer(validateRunConfigs),
		terraform.DynamicTransformer(func(g *terraform.Graph) error {
			cleanup := &TeardownSubgraph{opts: opts, parent: g, mode: b.CommandMode}
			g.Add(cleanup)

			// ensure that the teardown node runs after all the run nodes
			for v := range dag.ExcludeSeq[*TeardownSubgraph](g.VerticesSeq()) {
				if g.UpEdges(v).Len() == 0 {
					g.Connect(dag.BasicEdge(cleanup, v))
				}
			}

			return nil
		}),
		&TestProvidersTransformer{
			Config:    b.Config,
			File:      b.File,
			Providers: opts.ContextOpts.Providers,
		},
		&ReferenceTransformer{},
		&CloseTestGraphTransformer{},
		&terraform.TransitiveReductionTransformer{},
	}

	return steps
}

func validateRunConfigs(g *terraform.Graph) error {
	for node := range dag.SelectSeq[*NodeTestRun](g.VerticesSeq()) {
		diags := node.run.Config.Validate(node.run.ModuleConfig)
		node.run.Diagnostics = node.run.Diagnostics.Append(diags)
		if diags.HasErrors() {
			node.run.Status = moduletest.Error
		}
	}
	return nil
}

func Walk(g *terraform.Graph, ctx *EvalContext) tfdiags.Diagnostics {
	walkFn := func(v dag.Vertex) tfdiags.Diagnostics {
		if ctx.Cancelled() {
			// If the graph walk has been cancelled, the node should just return immediately.
			// For now, this means a hard stop has been requested, in this case we don't
			// even stop to mark future test runs as having been skipped. They'll
			// just show up as pending in the printed summary. We will quickly
			// just mark the overall file status has having errored to indicate
			// it was interrupted.
			return nil
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
			log.Printf("[TRACE] vertex %q: visit complete", dag.VertexName(v))
		}()

		// expandable nodes are not executed, but they are walked and
		// their children are executed, so they need not acquire the semaphore themselves.
		if _, ok := v.(Subgrapher); !ok {
			ctx.evalSem.Acquire()
			defer ctx.evalSem.Release()
		}

		if executable, ok := v.(GraphNodeExecutable); ok {
			executable.Execute(ctx)
		}
		return nil
	}

	return g.AcyclicGraph.Walk(walkFn)
}
