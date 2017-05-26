package structs

import (
	"fmt"
	"strings"

	"github.com/mitchellh/hashstructure"
)

const (
	// NodeUniqueNamespace is a prefix that can be appended to node meta or
	// attribute keys to mark them for exclusion in computed node class.
	NodeUniqueNamespace = "unique."
)

// UniqueNamespace takes a key and returns the key marked under the unique
// namespace.
func UniqueNamespace(key string) string {
	return fmt.Sprintf("%s%s", NodeUniqueNamespace, key)
}

// IsUniqueNamespace returns whether the key is under the unique namespace.
func IsUniqueNamespace(key string) bool {
	return strings.HasPrefix(key, NodeUniqueNamespace)
}

// ComputeClass computes a derived class for the node based on its attributes.
// ComputedClass is a unique id that identifies nodes with a common set of
// attributes and capabilities. Thus, when calculating a node's computed class
// we avoid including any uniquely identifing fields.
func (n *Node) ComputeClass() error {
	hash, err := hashstructure.Hash(n, nil)
	if err != nil {
		return err
	}

	n.ComputedClass = fmt.Sprintf("v1:%d", hash)
	return nil
}

// HashInclude is used to blacklist uniquely identifying node fields from being
// included in the computed node class.
func (n Node) HashInclude(field string, v interface{}) (bool, error) {
	switch field {
	case "Datacenter", "Attributes", "Meta", "NodeClass":
		return true, nil
	default:
		return false, nil
	}
}

// HashIncludeMap is used to blacklist uniquely identifying node map keys from being
// included in the computed node class.
func (n Node) HashIncludeMap(field string, k, v interface{}) (bool, error) {
	key, ok := k.(string)
	if !ok {
		return false, fmt.Errorf("map key %v not a string", k)
	}

	switch field {
	case "Meta", "Attributes":
		return !IsUniqueNamespace(key), nil
	default:
		return false, fmt.Errorf("unexpected map field: %v", field)
	}
}

// EscapedConstraints takes a set of constraints and returns the set that
// escapes computed node classes.
func EscapedConstraints(constraints []*Constraint) []*Constraint {
	var escaped []*Constraint
	for _, c := range constraints {
		if constraintTargetEscapes(c.LTarget) || constraintTargetEscapes(c.RTarget) {
			escaped = append(escaped, c)
		}
	}

	return escaped
}

// constraintTargetEscapes returns whether the target of a constraint escapes
// computed node class optimization.
func constraintTargetEscapes(target string) bool {
	switch {
	case strings.HasPrefix(target, "${node.unique."):
		return true
	case strings.HasPrefix(target, "${attr.unique."):
		return true
	case strings.HasPrefix(target, "${meta.unique."):
		return true
	default:
		return false
	}
}
