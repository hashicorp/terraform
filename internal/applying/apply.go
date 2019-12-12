package applying

import (
	"context"
	"log"

	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// apply is the real internal implementation of Apply, which can assume the
// arguments it recieves are valid and complete.
func apply(ctx context.Context, args Arguments) (*states.State, tfdiags.Diagnostics) {
	state := args.PriorState.DeepCopy()
	var diags tfdiags.Diagnostics

	graph, moreDiags := buildGraph(
		args.PriorState,
		args.Config,
		args.Plan,
		args.Schemas,
	)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return state, diags
	}

	log.Printf("[TRACE] Apply: action dependency graph:\n%s", logging.Indent(graph.StringWithNodeTypes()))

	// Actions will mutate the state via the data object during their work.
	data := newActionData(args.Dependencies, args.Schemas, state)

	moreDiags = graph.Walk(func(v dag.Vertex) tfdiags.Diagnostics {
		log.Printf("[TRACE] Apply: starting %q", dag.VertexName(v))
		diags := v.(action).Execute(ctx, data)
		log.Printf("[TRACE] Apply: finished %q", dag.VertexName(v))
		return diags
	})
	diags = diags.Append(moreDiags)

	// The actions mutate our state object directly as they work, so once
	// we get here "state" represents the result of the apply operation, even
	// if the walk was aborted early due to errors.

	return state, diags
}
