package dynblock

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

// WalkVariables begins the recursive process of walking all expressions and
// nested blocks in the given body and its child bodies while taking into
// account any "dynamic" blocks.
//
// This function requires that the caller walk through the nested block
// structure in the given body level-by-level so that an appropriate schema
// can be provided at each level to inform further processing. This workflow
// is thus easiest to use for calling applications that have some higher-level
// schema representation available with which to drive this multi-step
// process. If your application uses the hcldec package, you may be able to
// use VariablesHCLDec instead for a more automatic approach.
func WalkVariables(body hcl.Body) WalkVariablesNode {
	return WalkVariablesNode{
		body:           body,
		includeContent: true,
	}
}

// WalkExpandVariables is like Variables but it includes only the variables
// required for successful block expansion, ignoring any variables referenced
// inside block contents. The result is the minimal set of all variables
// required for a call to Expand, excluding variables that would only be
// needed to subsequently call Content or PartialContent on the expanded
// body.
func WalkExpandVariables(body hcl.Body) WalkVariablesNode {
	return WalkVariablesNode{
		body: body,
	}
}

type WalkVariablesNode struct {
	body hcl.Body
	it   *iteration

	includeContent bool
}

type WalkVariablesChild struct {
	BlockTypeName string
	Node          WalkVariablesNode
}

// Body returns the HCL Body associated with the child node, in case the caller
// wants to do some sort of inspection of it in order to decide what schema
// to pass to Visit.
//
// Most implementations should just fetch a fixed schema based on the
// BlockTypeName field and not access this. Deciding on a schema dynamically
// based on the body is a strange thing to do and generally necessary only if
// your caller is already doing other bizarre things with HCL bodies.
func (c WalkVariablesChild) Body() hcl.Body {
	return c.Node.body
}

// Visit returns the variable traversals required for any "dynamic" blocks
// directly in the body associated with this node, and also returns any child
// nodes that must be visited in order to continue the walk.
//
// Each child node has its associated block type name given in its BlockTypeName
// field, which the calling application should use to determine the appropriate
// schema for the content of each child node and pass it to the child node's
// own Visit method to continue the walk recursively.
func (n WalkVariablesNode) Visit(schema *hcl.BodySchema) (vars []hcl.Traversal, children []WalkVariablesChild) {
	extSchema := n.extendSchema(schema)
	container, _, _ := n.body.PartialContent(extSchema)
	if container == nil {
		return vars, children
	}

	children = make([]WalkVariablesChild, 0, len(container.Blocks))

	if n.includeContent {
		for _, attr := range container.Attributes {
			for _, traversal := range attr.Expr.Variables() {
				var ours, inherited bool
				if n.it != nil {
					ours = traversal.RootName() == n.it.IteratorName
					_, inherited = n.it.Inherited[traversal.RootName()]
				}

				if !(ours || inherited) {
					vars = append(vars, traversal)
				}
			}
		}
	}

	for _, block := range container.Blocks {
		switch block.Type {

		case "dynamic":
			blockTypeName := block.Labels[0]
			inner, _, _ := block.Body.PartialContent(variableDetectionInnerSchema)
			if inner == nil {
				continue
			}

			iteratorName := blockTypeName
			if attr, exists := inner.Attributes["iterator"]; exists {
				iterTraversal, _ := hcl.AbsTraversalForExpr(attr.Expr)
				if len(iterTraversal) == 0 {
					// Ignore this invalid dynamic block, since it'll produce
					// an error if someone tries to extract content from it
					// later anyway.
					continue
				}
				iteratorName = iterTraversal.RootName()
			}
			blockIt := n.it.MakeChild(iteratorName, cty.DynamicVal, cty.DynamicVal)

			if attr, exists := inner.Attributes["for_each"]; exists {
				// Filter out iterator names inherited from parent blocks
				for _, traversal := range attr.Expr.Variables() {
					if _, inherited := blockIt.Inherited[traversal.RootName()]; !inherited {
						vars = append(vars, traversal)
					}
				}
			}
			if attr, exists := inner.Attributes["labels"]; exists {
				// Filter out both our own iterator name _and_ those inherited
				// from parent blocks, since we provide _both_ of these to the
				// label expressions.
				for _, traversal := range attr.Expr.Variables() {
					ours := traversal.RootName() == iteratorName
					_, inherited := blockIt.Inherited[traversal.RootName()]

					if !(ours || inherited) {
						vars = append(vars, traversal)
					}
				}
			}

			for _, contentBlock := range inner.Blocks {
				// We only request "content" blocks in our schema, so we know
				// any blocks we find here will be content blocks. We require
				// exactly one content block for actual expansion, but we'll
				// be more liberal here so that callers can still collect
				// variables from erroneous "dynamic" blocks.
				children = append(children, WalkVariablesChild{
					BlockTypeName: blockTypeName,
					Node: WalkVariablesNode{
						body:           contentBlock.Body,
						it:             blockIt,
						includeContent: n.includeContent,
					},
				})
			}

		default:
			children = append(children, WalkVariablesChild{
				BlockTypeName: block.Type,
				Node: WalkVariablesNode{
					body:           block.Body,
					it:             n.it,
					includeContent: n.includeContent,
				},
			})

		}
	}

	return vars, children
}

func (n WalkVariablesNode) extendSchema(schema *hcl.BodySchema) *hcl.BodySchema {
	// We augment the requested schema to also include our special "dynamic"
	// block type, since then we'll get instances of it interleaved with
	// all of the literal child blocks we must also include.
	extSchema := &hcl.BodySchema{
		Attributes: schema.Attributes,
		Blocks:     make([]hcl.BlockHeaderSchema, len(schema.Blocks), len(schema.Blocks)+1),
	}
	copy(extSchema.Blocks, schema.Blocks)
	extSchema.Blocks = append(extSchema.Blocks, dynamicBlockHeaderSchema)

	return extSchema
}

// This is a more relaxed schema than what's in schema.go, since we
// want to maximize the amount of variables we can find even if there
// are erroneous blocks.
var variableDetectionInnerSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "for_each",
			Required: false,
		},
		{
			Name:     "labels",
			Required: false,
		},
		{
			Name:     "iterator",
			Required: false,
		},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "content",
		},
	},
}
