package terraform

import (
	"fmt"
)

// ImportStateTransformer is a GraphTransformer that adds nodes to the
// graph to represent the imports we want to do for resources.
type ImportStateTransformer struct {
	Targets []*ImportTarget
}

func (t *ImportStateTransformer) Transform(g *Graph) error {
	nodes := make([]*graphNodeImportState, 0, len(t.Targets))
	for _, target := range t.Targets {
		addr, err := ParseResourceAddress(target.Addr)
		if err != nil {
			return fmt.Errorf(
				"failed to parse resource address '%s': %s",
				target.Addr, err)
		}

		nodes = append(nodes, &graphNodeImportState{
			Addr: addr,
			ID:   target.ID,
		})
	}

	// Build the graph vertices
	for _, n := range nodes {
		g.Add(n)
	}

	return nil
}

type graphNodeImportState struct {
	Addr *ResourceAddress // Addr is the resource address to import to
	ID   string           // ID is the ID to import as
}

func (n *graphNodeImportState) Name() string {
	return fmt.Sprintf("import %s (id: %s)", n.Addr, n.ID)
}

func (n *graphNodeImportState) ProvidedBy() []string {
	return []string{resourceProvider(n.Addr.Type, "")}
}

// GraphNodeSubPath
func (n *graphNodeImportState) Path() []string {
	return n.Addr.Path
}

// GraphNodeEvalable impl.
func (n *graphNodeImportState) EvalTree() EvalNode {
	var provider ResourceProvider
	var states []*InstanceState
	info := &InstanceInfo{
		Id:         n.ID,
		ModulePath: n.Addr.Path,
		Type:       n.Addr.Type,
	}

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Name:   n.ProvidedBy()[0],
				Output: &provider,
			},
			&EvalImportState{
				Provider: &provider,
				Info:     info,
				Output:   &states,
			},
		},
	}
}
