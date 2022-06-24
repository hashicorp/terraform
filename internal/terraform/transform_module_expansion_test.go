package terraform

import (
	"testing"
)

var benchmarkModuleExpansionTransformerGraph *Graph

func BenchmarkModuleExpansionTransformer(b *testing.B) {
	// We need to construct quite an elaborate set of inputs in order to
	// make for a "realistic" run of the transformer that will generate
	// useful benchmark results. In particular, we need to have a
	// configuration with a relatively large number of non-root modules
	// so that the benchmark can be sensitive to the difference between
	// costs that scale per module or per level of nesting and costs
	// that are fixed regardless of the configuration tree complexity.

	cfg := testModule(b, "module-expansion-nesting")

	for i := 0; i < b.N; i++ {
		// We'll make sure that the graph "escapes" so that the Go compiler
		// can't optimize it away with local optimizations.
		benchmarkModuleExpansionTransformerGraph = func() *Graph {
			graph := &Graph{}

			// The module expansion transformer expects there to already be
			// graph nodes representing objects within the modules in the graph,
			// and so we'll borrow the ConfigTransformer to get an approximation
			// of that.
			cfgTransformer := &ConfigTransformer{
				Config: cfg,
			}
			cfgTransformer.Transform(graph)

			// Now we can run the module expansion transformer to add all of the
			// expand/close nodes for the modules and the edges from the expand
			// nodes to the resources inside.
			modExpTransformer := &ModuleExpansionTransformer{
				Config: cfg,
			}
			modExpTransformer.Transform(graph)

			return graph
		}()
	}

}
