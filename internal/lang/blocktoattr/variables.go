// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package blocktoattr

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/dynblock"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
)

// ExpandedVariables finds all of the global variables referenced in the
// given body with the given schema while taking into account the possibilities
// both of "dynamic" blocks being expanded and the possibility of certain
// attributes being written instead as nested blocks as allowed by the
// FixUpBlockAttrs function.
//
// This function exists to allow variables to be analyzed prior to dynamic
// block expansion while also dealing with the fact that dynamic block expansion
// might in turn produce nested blocks that are subject to FixUpBlockAttrs.
//
// This is intended as a drop-in replacement for dynblock.VariablesHCLDec,
// which is itself a drop-in replacement for hcldec.Variables.
func ExpandedVariables(body hcl.Body, schema *configschema.Block) []hcl.Traversal {
	rootNode := dynblock.WalkVariables(body)
	return walkVariables(rootNode, body, schema)
}

func walkVariables(node dynblock.WalkVariablesNode, body hcl.Body, schema *configschema.Block) []hcl.Traversal {
	givenRawSchema := hcldec.ImpliedSchema(schema.DecoderSpec())
	ambiguousNames := ambiguousNames(schema)
	effectiveRawSchema := effectiveSchema(givenRawSchema, body, ambiguousNames, false)
	vars, children := node.Visit(effectiveRawSchema)

	for _, child := range children {
		if blockS, exists := schema.BlockTypes[child.BlockTypeName]; exists {
			vars = append(vars, walkVariables(child.Node, child.Body(), &blockS.Block)...)
		} else if attrS, exists := schema.Attributes[child.BlockTypeName]; exists && attrS.Type.IsCollectionType() && attrS.Type.ElementType().IsObjectType() {
			// ☝️Check for collection type before element type, because if this is a mis-placed reference,
			// a panic here will prevent other useful diags from being elevated to show the user what to fix
			synthSchema := SchemaForCtyElementType(attrS.Type.ElementType())
			vars = append(vars, walkVariables(child.Node, child.Body(), synthSchema)...)
		}
	}

	return vars
}

type InstanceTraversal struct {
	Key       hcl.Expression
	Reference *addrs.Reference

	// The key of the collection may also be an instance traversal
	KeyAsInstanceTraversal *InstanceTraversal

	AllKeys bool
}

// InstanceTraversals returns a list of resource instance traversals within the given body and schema.
func InstanceTraversals(body hcl.Body, schema *configschema.Block) []*InstanceTraversal {
	rootNode := dynblock.WalkVariables(body)
	ret := walkVariables2(rootNode, body, schema)
	return ret
}

func ExprInstanceTraversals(rawExpr hcl.Expression) []*InstanceTraversal {
	ret := []*InstanceTraversal{}
	if rawExpr == nil {
		return ret
	}
	if tr := IndexToInstanceTraversal(rawExpr); tr != nil {
		ret = append(ret, tr)
	}

	children := rawExpr.Children()
	for _, expr := range children {
		ret = append(ret, ExprInstanceTraversals(expr)...)
	}
	return ret
}

func walkVariables2(node dynblock.WalkVariablesNode, body hcl.Body, schema *configschema.Block) []*InstanceTraversal {
	ret := []*InstanceTraversal{}
	givenRawSchema := hcldec.ImpliedSchema(schema.DecoderSpec())
	ambiguousNames := ambiguousNames(schema)
	effectiveRawSchema := effectiveSchema(givenRawSchema, body, ambiguousNames, false)
	_, children := node.Visit(effectiveRawSchema)
	container, _, _ := body.PartialContent(effectiveRawSchema)
	if container == nil {
		return ret
	}
	for _, attr := range container.Attributes {
		ret = append(ret, ExprInstanceTraversals(attr.Expr)...)
	}

	for _, child := range children {
		if blockS, exists := schema.BlockTypes[child.BlockTypeName]; exists {
			ret = append(ret, walkVariables2(child.Node, child.Body(), &blockS.Block)...)
		} else if attrS, exists := schema.Attributes[child.BlockTypeName]; exists && attrS.Type.IsCollectionType() && attrS.Type.ElementType().IsObjectType() {
			// ☝️Check for collection type before element type, because if this is a mis-placed reference,
			// a panic here will prevent other useful diags from being elevated to show the user what to fix
			synthSchema := SchemaForCtyElementType(attrS.Type.ElementType())
			ret = append(ret, walkVariables2(child.Node, child.Body(), synthSchema)...)
		}
	}

	return ret
}

func IndexToInstanceTraversal(rawExpr hcl.Expression) *InstanceTraversal {
	var indexExpr *hclsyntax.IndexExpr
	var traversal hcl.Traversal
	if expr, ok := rawExpr.(*hclsyntax.IndexExpr); ok {
		indexExpr = expr
		collection := indexExpr.Collection.Variables()
		if collection == nil {
			return nil
		}
		traversal = collection[0]
	} else if expr, ok := rawExpr.(*hclsyntax.RelativeTraversalExpr); ok {
		if expr, ok := expr.Source.(*hclsyntax.IndexExpr); ok {
			indexExpr = expr
			collection := indexExpr.Collection.Variables()
			if collection != nil {
				traversal = collection[0]
			}
		}
	} else if expr, ok := rawExpr.(*hclsyntax.SplatExpr); ok {
		collection := expr.Source.Variables()
		if collection != nil {
			traversal = collection[0]
		}
	}

	if traversal == nil {
		return nil
	}

	ref, diags := addrs.ParseRef(traversal)
	if diags.HasErrors() {
		return nil
	}

	if ref.Subject == nil {
		return nil
	}

	if _, ok := ref.Subject.(addrs.Resource); ok {
		if indexExpr != nil {
			return &InstanceTraversal{
				Key:                    indexExpr.Key,
				Reference:              ref,
				KeyAsInstanceTraversal: IndexToInstanceTraversal(indexExpr.Key),
			}
		}

		return &InstanceTraversal{
			Reference: ref,
			AllKeys:   true,
		}
	}

	return nil
}
