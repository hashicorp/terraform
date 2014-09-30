// The depgraph package is used to create and model a dependency graph
// of nouns. Each noun can represent a service, server, application,
// network switch, etc. Nouns can depend on other nouns, and provide
// versioning constraints. Nouns can also have various meta data that
// may be relevant to their construction or configuration.
package depgraph

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/digraph"
)

// WalkFunc is the type used for the callback for Walk.
type WalkFunc func(*Noun) error

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
	var msgs []string

	if v.MissingRoot {
		msgs = append(msgs, "The graph has no single root")
	}

	for _, n := range v.Unreachable {
		msgs = append(msgs, fmt.Sprintf(
			"Unreachable node: %s", n.Name))
	}

	for _, c := range v.Cycles {
		cycleNodes := make([]string, len(c))
		for i, n := range c {
			cycleNodes[i] = n.Name
		}

		msgs = append(msgs, fmt.Sprintf(
			"Cycle: %s", strings.Join(cycleNodes, " -> ")))
	}

	for i, m := range msgs {
		msgs[i] = fmt.Sprintf("* %s", m)
	}

	return fmt.Sprintf(
		"The dependency graph is not valid:\n\n%s",
		strings.Join(msgs, "\n"))
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

// Noun returns the noun with the given name, or nil if it cannot be found.
func (g *Graph) Noun(name string) *Noun {
	for _, n := range g.Nouns {
		if n.Name == name {
			return n
		}
	}

	return nil
}

// String generates a little ASCII string of the graph, useful in
// debugging output.
func (g *Graph) String() string {
	var buf bytes.Buffer

	// Alphabetize the output based on the noun name
	keys := make([]string, 0, len(g.Nouns))
	mapping := make(map[string]*Noun)
	for _, n := range g.Nouns {
		mapping[n.Name] = n
		keys = append(keys, n.Name)
	}
	sort.Strings(keys)

	if g.Root != nil {
		buf.WriteString(fmt.Sprintf("root: %s\n", g.Root.Name))
	} else {
		buf.WriteString("root: <unknown>\n")
	}
	for _, k := range keys {
		n := mapping[k]
		buf.WriteString(fmt.Sprintf("%s\n", n.Name))

		// Alphabetize the dependency names
		depKeys := make([]string, 0, len(n.Deps))
		depMapping := make(map[string]*Dependency)
		for _, d := range n.Deps {
			depMapping[d.Target.Name] = d
			depKeys = append(depKeys, d.Target.Name)
		}
		sort.Strings(depKeys)

		for _, k := range depKeys {
			dep := depMapping[k]
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

	// Check for loops to yourself
	for _, n := range g.Nouns {
		for _, d := range n.Deps {
			if d.Source == d.Target {
				vErr.Cycles = append(vErr.Cycles, []*Noun{n})
			}
		}
	}

	// Return the detailed error
	if vErr.MissingRoot || vErr.Unreachable != nil || vErr.Cycles != nil {
		return vErr
	}
	return nil
}

// Walk will walk the tree depth-first (dependency first) and call
// the callback.
//
// The callbacks will be called in parallel, so if you need non-parallelism,
// then introduce a lock in your callback.
func (g *Graph) Walk(fn WalkFunc) error {
	// Set so we don't callback for a single noun multiple times
	var seenMapL sync.RWMutex
	seenMap := make(map[*Noun]chan struct{})
	seenMap[g.Root] = make(chan struct{})

	// Keep track of what nodes errored.
	var errMapL sync.RWMutex
	errMap := make(map[*Noun]struct{})

	// Build the list of things to visit
	tovisit := make([]*Noun, 1, len(g.Nouns))
	tovisit[0] = g.Root

	// Spawn off all our goroutines to walk the tree
	errCh := make(chan error)
	for len(tovisit) > 0 {
		// Grab the current thing to use
		n := len(tovisit)
		current := tovisit[n-1]
		tovisit = tovisit[:n-1]

		// Go through each dependency and run that first
		for _, dep := range current.Deps {
			if _, ok := seenMap[dep.Target]; !ok {
				seenMapL.Lock()
				seenMap[dep.Target] = make(chan struct{})
				seenMapL.Unlock()
				tovisit = append(tovisit, dep.Target)
			}
		}

		// Spawn off a goroutine to execute our callback once
		// all our dependencies are satisfied.
		go func(current *Noun) {
			seenMapL.RLock()
			closeCh := seenMap[current]
			seenMapL.RUnlock()

			defer close(closeCh)

			// Wait for all our dependencies
			for _, dep := range current.Deps {
				seenMapL.RLock()
				ch := seenMap[dep.Target]
				seenMapL.RUnlock()

				// Wait for the dep to be run
				<-ch

				// Check if any dependencies errored. If so,
				// then return right away, we won't walk it.
				errMapL.RLock()
				_, errOk := errMap[dep.Target]
				errMapL.RUnlock()
				if errOk {
					return
				}
			}

			// Call our callback!
			if err := fn(current); err != nil {
				errMapL.Lock()
				errMap[current] = struct{}{}
				errMapL.Unlock()

				errCh <- err
			}
		}(current)
	}

	// Aggregate channel that is closed when all goroutines finish
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)

		for _, ch := range seenMap {
			<-ch
		}
	}()

	// Wait for finish OR an error
	select {
	case <-doneCh:
		return nil
	case err := <-errCh:
		// Drain the error channel
		go func() {
			for _ = range errCh {
				// Nothing
			}
		}()

		// Wait for the goroutines to end
		<-doneCh
		close(errCh)

		return err
	}
}

// DependsOn returns the set of nouns that have a
// dependency on a given noun. This can be used to find
// the incoming edges to a noun.
func (g *Graph) DependsOn(n *Noun) []*Noun {
	var incoming []*Noun
OUTER:
	for _, other := range g.Nouns {
		if other == n {
			continue
		}
		for _, d := range other.Deps {
			if d.Target == n {
				incoming = append(incoming, other)
				continue OUTER
			}
		}
	}
	return incoming
}
