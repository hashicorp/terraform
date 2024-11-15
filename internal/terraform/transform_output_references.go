// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

// OutputReferencesTransformer is a GraphTransformer that adds meta-information
// about references to output values in the configuration. This is used to
// determine when deprecated outputs are used.
type OutputReferencesTransformer struct{}

func (t *OutputReferencesTransformer) Transform(g *Graph) error {
	// Build a reference map so we can efficiently look up the references
	vs := g.Vertices()
	m := NewReferenceMap(vs)

	// Find the things that reference things and connect them
	for _, v := range vs {
		if dependant, ok := v.(GraphNodeReferencer); ok {
			parents := m.References(v)

			for _, parent := range parents {
				if output, ok := parent.(*nodeExpandOutput); ok {
					output.Dependants = append(output.Dependants, dependant.References()...)
				}
			}
		}
	}

	return nil
}
