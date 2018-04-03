package dynblock

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcldec"
)

// ForEachVariablesHCLDec is a wrapper around WalkForEachVariables that
// uses the given hcldec specification to automatically drive the recursive
// walk through nested blocks in the given body.
//
// This provides more convenient access to all of the "for_each" and "labels"
// dependencies in a body for applications that are already using hcldec
// as a more convenient way to recursively decode body contents.
func ForEachVariablesHCLDec(body hcl.Body, spec hcldec.Spec) []hcl.Traversal {
	rootNode := WalkForEachVariables(body)
	return walkVariablesWithHCLDec(rootNode, spec)
}

func walkVariablesWithHCLDec(node WalkVariablesNode, spec hcldec.Spec) []hcl.Traversal {
	vars, children := node.Visit(hcldec.ImpliedSchema(spec))

	if len(children) > 0 {
		childSpecs := hcldec.ChildBlockTypes(spec)
		for _, child := range children {
			if childSpec, exists := childSpecs[child.BlockTypeName]; exists {
				vars = append(vars, walkVariablesWithHCLDec(child.Node, childSpec)...)
			}
		}
	}

	return vars
}
