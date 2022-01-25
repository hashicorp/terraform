package terraform

// TransitiveReductionTransformer is a GraphTransformer that performs
// finds the transitive reduction of the graph. For a definition of
// transitive reduction, see Wikipedia.
type TransitiveReductionTransformer struct{}

func (t *TransitiveReductionTransformer) Transform(g *Graph) error {
	// If the graph isn't valid, skip the transitive reduction.
	// We don't error here because Terraform itself handles graph
	// validation in a better way, or we assume it does.
	if err := g.Validate(); err != nil {
		return nil
	}

	// Do it
	g.TransitiveReduction()

	return nil
}
