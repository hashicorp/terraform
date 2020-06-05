package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/configs"
)

// ModuleVariableTransformer is a GraphTransformer that adds all the variables
// in the configuration to the graph.
//
// Any "variable" block present in any non-root module is included here, even
// if a particular variable is not referenced from anywhere.
//
// The transform will produce errors if a call to a module does not conform
// to the expected set of arguments, but this transformer is not in a good
// position to return errors and so the validate walk should include specific
// steps for validating module blocks, separate from this transform.
type ModuleVariableTransformer struct {
	Config *configs.Config
}

func (t *ModuleVariableTransformer) Transform(g *Graph) error {
	return t.transform(g, nil, t.Config)
}

func (t *ModuleVariableTransformer) transform(g *Graph, parent, c *configs.Config) error {
	// We can have no variables if we have no configuration.
	if c == nil {
		return nil
	}

	// Transform all the children first.
	for _, cc := range c.Children {
		if err := t.transform(g, c, cc); err != nil {
			return err
		}
	}

	// If we're processing anything other than the root module then we'll
	// add graph nodes for variables defined inside. (Variables for the root
	// module are dealt with in RootVariableTransformer).
	// If we have a parent, we can determine if a module variable is being
	// used, so we transform this.
	if parent != nil {
		if err := t.transformSingle(g, parent, c); err != nil {
			return err
		}
	}

	return nil
}

func (t *ModuleVariableTransformer) transformSingle(g *Graph, parent, c *configs.Config) error {
	_, call := c.Path.Call()

	// Find the call in the parent module configuration, so we can get the
	// expressions given for each input variable at the call site.
	callConfig, exists := parent.Module.ModuleCalls[call.Name]
	if !exists {
		// This should never happen, since it indicates an improperly-constructed
		// configuration tree.
		panic(fmt.Errorf("no module call block found for %s", c.Path))
	}

	// We need to construct a schema for the expected call arguments based on
	// the configured variables in our config, which we can then use to
	// decode the content of the call block.
	schema := &hcl.BodySchema{}
	for _, v := range c.Module.Variables {
		schema.Attributes = append(schema.Attributes, hcl.AttributeSchema{
			Name:     v.Name,
			Required: v.Default == cty.NilVal,
		})
	}

	content, contentDiags := callConfig.Config.Content(schema)
	if contentDiags.HasErrors() {
		// Validation code elsewhere should deal with any errors before we
		// get in here, but we'll report them out here just in case, to
		// avoid crashes.
		var diags tfdiags.Diagnostics
		diags = diags.Append(contentDiags)
		return diags.Err()
	}

	for _, v := range c.Module.Variables {
		var expr hcl.Expression
		if attr := content.Attributes[v.Name]; attr != nil {
			expr = attr.Expr
		} else {
			// No expression provided for this variable, so we'll make a
			// synthetic one using the variable's default value.
			expr = &hclsyntax.LiteralValueExpr{
				Val:      v.Default,
				SrcRange: v.DeclRange, // This is not exact, but close enough
			}
		}

		// Add a plannable node, as the variable may expand
		// during module expansion
		node := &nodeExpandModuleVariable{
			Addr: addrs.InputVariable{
				Name: v.Name,
			},
			Module: c.Path,
			Config: v,
			Expr:   expr,
		}
		g.Add(node)
	}

	return nil
}
