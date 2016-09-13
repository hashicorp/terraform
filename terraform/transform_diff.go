package terraform

import (
	"fmt"
)

// DiffTransformer is a GraphTransformer that adds the elements of
// the diff to the graph.
//
// This transform is used for example by the ApplyGraphBuilder to ensure
// that only resources that are being modified are represented in the graph.
type DiffTransformer struct {
	Diff *Diff
}

func (t *DiffTransformer) Transform(g *Graph) error {
	// If the diff is nil or empty (nil is empty) then do nothing
	if t.Diff.Empty() {
		return nil
	}

	// Go through all the modules in the diff.
	for _, m := range t.Diff.Modules {
		// TODO: If this is a destroy diff then add a module destroy node

		// Go through all the resources in this module.
		for name, inst := range m.Resources {
			// TODO: Destroy diff

			// If this diff has no attribute changes, then we have
			// nothing to do and therefore won't add it to the graph.
			if len(inst.Attributes) == 0 {
				continue
			}

			// We have changes! This is a create or update operation.
			// First grab the address so we have a unique way to
			// reference this resource.
			addr, err := parseResourceAddressInternal(name)
			if err != nil {
				return fmt.Errorf(
					"Error parsing internal name, this is a bug: %q", name)
			}

			// Add the resource to the graph
			g.Add(&NodeResource{
				Addr: addr,
			})
		}
	}

	return nil
}
