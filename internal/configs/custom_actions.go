package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
)

type CustomActionStep struct {
	Name string

	// Receiver is the address of the object that the action should be invoked
	// on, or nil if the action should be invoked on the module instance
	// where the step is declared.
	//
	// In practice only a subset of referencable expression types are allowed
	// here, but that's enforced by the runtime rather than by the configuration
	// decoder.
	Receiver addrs.Referenceable

	// ActionTypeName is the name of the action to invoke. This should refer
	// to an action type that is defined for the given receiver, but that's
	// checked by the runtime rather than by the configuration decoder.
	ActionTypeName string

	// Arguments is the not-yet-decoded body of arguments for the action
	// invocation.
	Arguments hcl.Body

	DeclRange      hcl.Range
	ArgumentsRange hcl.Range
}

type CustomActionSequence struct {
	Steps []*CustomActionStep

	DeclRange hcl.Range
}

type ModuleDefinedCustomAction struct {
	Name string

	Variables map[string]*Variable
	Steps     []*CustomActionStep
	Outputs   map[string]*Output

	DeclRange hcl.Range
}

func decodeCustomActionSequenceBlock(block *hcl.Block) (*CustomActionSequence, hcl.Diagnostics) {
	steps, diags := decodeCustomActionSequenceBody(block.Body)
	return &CustomActionSequence{
		Steps:     steps,
		DeclRange: block.DefRange,
	}, diags
}

func decodeModuleDefinedCustomActionBlock(block *hcl.Block) (*ModuleDefinedCustomAction, hcl.Diagnostics) {
	content, remain, diags := block.Body.PartialContent(moduleDefinedCustomActionSchema)

	name := block.Labels[0]
	if !hclsyntax.ValidIdentifier(name) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid custom action name",
			Detail:   "Custom action names must be valid identifiers.",
			Subject:  &block.LabelRanges[0],
		})
	}

	steps, moreDiags := decodeCustomActionSequenceBody(remain)
	diags = append(diags, moreDiags...)

	variables := make(map[string]*Variable)
	outputs := make(map[string]*Output)
	for _, block := range content.Blocks {
		switch block.Type {
		case "variable":
			v, moreDiags := decodeVariableBlock(block, false)
			diags = append(diags, moreDiags...)
			if v == nil {
				continue
			}
			if existing, exists := variables[v.Name]; exists {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate custom action input variable declaration",
					Detail: fmt.Sprintf(
						"This custom action already has an input variable named %q, declared at %s.",
						existing.Name, existing.DeclRange,
					),
				})
				continue
			}
			variables[v.Name] = v
		case "output":
			o, moreDiags := decodeOutputBlock(block, false)
			diags = append(diags, moreDiags...)
			if o == nil {
				continue
			}
			if existing, exists := outputs[o.Name]; exists {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate custom action output value declaration",
					Detail: fmt.Sprintf(
						"This custom action already has an output value named %q, declared at %s.",
						existing.Name, existing.DeclRange,
					),
				})
				continue
			}
			outputs[o.Name] = o
		default:
			// We should not get here because there are no other block types
			// declared in our schema.
		}
	}

	return &ModuleDefinedCustomAction{
		Variables: variables,
		Steps:     steps,
		Outputs:   outputs,
		DeclRange: block.DefRange,
	}, diags
}

func decodeCustomActionSequenceBody(body hcl.Body) ([]*CustomActionStep, hcl.Diagnostics) {
	var steps []*CustomActionStep

	content, diags := body.Content(customActionSequenceSchema)
	if diags.HasErrors() {
		return steps, diags
	}

	for _, block := range content.Blocks {
		switch block.Type {
		case "step":
			step, moreDiags := decodeCustomActionStepBlock(block)
			diags = append(diags, moreDiags...)
			if step != nil {
				steps = append(steps, step)
			}
		default:
			// We should not get here because there are no other block types
			// declared in our schema.
		}
	}

	return steps, diags
}

func decodeCustomActionStepBlock(block *hcl.Block) (*CustomActionStep, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	name := block.Labels[0]
	if !hclsyntax.ValidIdentifier(name) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid custom action step name",
			Detail:   "Custom action step names must be valid identifiers.",
			Subject:  &block.LabelRanges[0],
		})
	}

	content, moreDiags := block.Body.Content(customActionStepSchema)
	diags = append(diags, moreDiags...)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	ret := &CustomActionStep{
		Name:      name,
		DeclRange: block.DefRange,
	}

	if attr, ok := content.Attributes["receiver"]; ok {
		traversal, moreDiags := hcl.AbsTraversalForExpr(attr.Expr)
		diags = append(diags, moreDiags...)
		if moreDiags.HasErrors() {
			return nil, diags
		}
		ref, refDiags := addrs.ParseRef(traversal)
		diags = append(diags, refDiags.ToHCL()...)
		if moreDiags.HasErrors() {
			return nil, diags
		}
		ret.Receiver = ref.Subject
	}

	if attr, ok := content.Attributes["action"]; ok {
		name := hcl.ExprAsKeyword(attr.Expr)
		if name == "" {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid action type reference",
				Detail:   "Must be an identifier referring to a custom action action that is defined for the object named in 'receiver'.",
				Subject:  attr.Expr.Range().Ptr(),
			})
			return nil, diags
		}
		ret.ActionTypeName = name
	}

	for _, block := range content.Blocks {
		switch block.Type {
		case "arguments":
			if ret.Arguments != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate arguments block for custom action step",
					Detail:   fmt.Sprintf("This custom action step already has its arguments defined at %s.", ret.ArgumentsRange),
					Subject:  block.TypeRange.Ptr(),
				})
				continue
			}
		default:
			// We should not get here because there are no other block types
			// declared in our schema.
		}
	}

	// If there was no "arguments" block then we'll provide an empty body
	// just to make life easier for downstream code that's consuming this
	// result.
	if ret.Arguments == nil {
		ret.Arguments = hcl.EmptyBody()
		ret.ArgumentsRange = ret.DeclRange // close enough for diagnostics
	}

	return ret, diags
}

var customActionSequenceSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "step", LabelNames: []string{"name"}},
	},
}

var customActionStepSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "receiver", Required: false},
		{Name: "action", Required: true},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "arguments"},
	},
}

var moduleDefinedCustomActionSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "variable", LabelNames: []string{"name"}},
		{Type: "output", LabelNames: []string{"name"}},

		// (This should be decoded using PartialContent and then the
		// remainder used with customActionSequenceSchema.)
	},
}
