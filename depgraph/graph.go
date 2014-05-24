// The depgraph package is used to create and model a dependency graph
// of nouns. Each noun can represent a service, server, application,
// network switch, etc. Nouns can depend on other nouns, and provide
// versioning constraints. Nouns can also have various meta data that
// may be relevant to their construction or configuration.
package depgraph

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform/digraph"
)

// Graph is used to represent a dependency graph.
type Graph struct {
	Name  string
	Meta  interface{}
	Nouns []*Noun
	Root  *Noun
}

// ValidateError implements the Error interface but provides
// additional information on a validation error.
type ValidateError struct {
	// If set, then the graph is missing a single root, on which
	// there are no depdendencies
	MissingRoot bool

	// Unreachable are nodes that could not be reached from
	// the root noun.
	Unreachable []*Noun

	// Cycles are groups of strongly connected nodes, which
	// form a cycle. This is disallowed.
	Cycles [][]*Noun
}

func (v *ValidateError) Error() string {
	return "The depedency graph is not valid"
}

// ConstraintError is used to return detailed violation
// information from CheckConstraints
type ConstraintError struct {
	Violations []*Violation
}

func (c *ConstraintError) Error() string {
	return fmt.Sprintf("%d constraint violations", len(c.Violations))
}

// Violation is used to pass along information about
// a constraint violation
type Violation struct {
	Source     *Noun
	Target     *Noun
	Dependency *Dependency
	Constraint Constraint
	Err        error
}

func (v *Violation) Error() string {
	return fmt.Sprintf("Constraint %v between %v and %v violated: %v",
		v.Constraint, v.Source, v.Target, v.Err)
}

// CheckConstraints walks the graph and ensures that all
// user imposed constraints are satisfied.
func (g *Graph) CheckConstraints() error {
	// Ensure we have a root
	if g.Root == nil {
		return fmt.Errorf("Graph must be validated before checking constraint violations")
	}

	// Create a constraint error
	cErr := &ConstraintError{}

	// Walk from the root
	digraph.DepthFirstWalk(g.Root, func(n digraph.Node) bool {
		noun := n.(*Noun)
		for _, dep := range noun.Deps {
			target := dep.Target
			for _, constraint := range dep.Constraints {
				ok, err := constraint.Satisfied(noun, target)
				if ok {
					continue
				}
				violation := &Violation{
					Source:     noun,
					Target:     target,
					Dependency: dep,
					Constraint: constraint,
					Err:        err,
				}
				cErr.Violations = append(cErr.Violations, violation)
			}
		}
		return true
	})

	if cErr.Violations != nil {
		return cErr
	}
	return nil
}

// String generates a little ASCII string of the graph, useful in
// debugging output.
func (g *Graph) String() string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("root: %s\n", g.Root.Name))
	for _, dep := range g.Root.Deps {
		buf.WriteString(fmt.Sprintf(
			"  %s -> %s\n",
			dep.Source,
			dep.Target))
		}

	for _, n := range g.Nouns {
		buf.WriteString(fmt.Sprintf("%s\n", n.Name))
		for _, dep := range n.Deps {
			buf.WriteString(fmt.Sprintf(
				"  %s -> %s\n",
				dep.Source,
				dep.Target))
		}
	}

	return buf.String()
}

// Validate is used to ensure that a few properties of the graph are not violated:
// 1) There must be a single "root", or source on which nothing depends.
// 2) All nouns in the graph must be reachable from the root
// 3) The graph must be cycle free, meaning there are no cicular dependencies
func (g *Graph) Validate() error {
	// Convert to node list
	nodes := make([]digraph.Node, len(g.Nouns))
	for i, n := range g.Nouns {
		nodes[i] = n
	}

	// Create a validate erro
	vErr := &ValidateError{}

	// Search for all the sources, if we have only 1, it must be the root
	if sources := digraph.Sources(nodes); len(sources) != 1 {
		vErr.MissingRoot = true
		goto CHECK_CYCLES
	} else {
		g.Root = sources[0].(*Noun)
	}

	// Check reachability
	if unreached := digraph.Unreachable(g.Root, nodes); len(unreached) > 0 {
		vErr.Unreachable = make([]*Noun, len(unreached))
		for i, u := range unreached {
			vErr.Unreachable[i] = u.(*Noun)
		}
	}

CHECK_CYCLES:
	// Check for cycles
	if cycles := digraph.StronglyConnectedComponents(nodes, true); len(cycles) > 0 {
		vErr.Cycles = make([][]*Noun, len(cycles))
		for i, cycle := range cycles {
			group := make([]*Noun, len(cycle))
			for j, n := range cycle {
				group[j] = n.(*Noun)
			}
			vErr.Cycles[i] = group
		}
	}

	// Return the detailed error
	if vErr.MissingRoot || vErr.Unreachable != nil || vErr.Cycles != nil {
		return vErr
	}
	return nil
}
