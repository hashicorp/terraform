// Package tfcomponents implements the parser, decoder, and static in-memory
// representation of the ".tfcomponents.hcl" language used to define a set
// of components, where each component is a tree of modules that is operated
// on as a unit in an execution context isolated (to some extent) from all
// others.
package tfcomponents
