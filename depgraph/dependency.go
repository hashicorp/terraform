package depgraph

import (
	"github.com/hashicorp/terraform/digraph"
)

// Dependency is used to create a directed edge between two nouns.
// One noun may depend on another and provide version constraints
// that cannot be violated
type Dependency struct {
	Name        string
	Meta        interface{}
	Constraints []Constraint
	Source      *Noun
	Target      *Noun
}

// Constraint is used by dependencies to allow arbitrary constraints
// between nouns
type Constraint interface {
	Satisfied(head, tail *Noun) (bool, error)
}

// Head returns the source, or dependent noun
func (d *Dependency) Head() digraph.Node {
	return d.Source
}

// Tail returns the target, or depended upon noun
func (d *Dependency) Tail() digraph.Node {
	return d.Target
}

func (d *Dependency) String() string {
	return d.Name
}
