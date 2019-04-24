package dynblock

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcldec"
)

// VariablesHCLDec is a wrapper around WalkVariables that uses the given hcldec
// specification to automatically drive the recursive walk through nested
// blocks in the given body.
//
// This is a drop-in replacement for hcldec.Variables which is able to treat
// blocks of type "dynamic" in the same special way that dynblock.Expand would,
// exposing both the variables referenced in the "for_each" and "labels"
// arguments and variables used in the nested "content" block.
func VariablesHCLDec(body hcl.Body, spec hcldec.Spec) []hcl.Traversal {
	rootNode := WalkVariables(body)
	return walkVariablesWithHCLDec(rootNode, spec)
}

// ExpandVariablesHCLDec is like VariablesHCLDec but it includes only the
// minimal set of variables required to call Expand, ignoring variables that
// are referenced only inside normal block contents. See WalkExpandVariables
// for more information.
func ExpandVariablesHCLDec(body hcl.Body, spec hcldec.Spec) []hcl.Traversal {
	rootNode := WalkExpandVariables(body)
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
