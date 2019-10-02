package projectconfigs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/tfdiags"
)

// Workspace represents a workspace configuration block.
//
// Note that a single workspace configuration block might actually expand to
// multiple workspace definitions, if ForEach is non-nil.
type Workspace struct {
	// Name is the name label given in the block header.
	//
	// It is guaranteed to be a valid HCL identifier.
	Name string

	// ForEach is the expression given in the for_each argument, or nil if
	// that argument wasn't set.
	ForEach hcl.Expression

	// Variables is the expression given in the "variables" argument, or nil
	// if that argument wasn't set.
	Variables hcl.Expression

	// ConfigSource is the expression representing the location of the root
	// module of the configuration for this workspace, relative to the
	// project root.
	ConfigSource hcl.Expression

	// StateStorage represents the contents of a state_storage block, or nil
	// for a remote workspace.
	//
	// StateStorage and Remote are mutually exclusive
	StateStorage *StateStorage

	// Remote is the expression given in the "remote" argument for a remote
	// workspace, or nil for local workspaces.
	//
	// Remote and StateStorage are mutually exclusive.
	Remote hcl.Expression

	// DeclRange is the source range of the block header of this block,
	// for use in diagnostic messages. NameRange is the range of the
	// Name string specifically.
	DeclRange, NameRange tfdiags.SourceRange
}

func decodeWorkspaceBlock(block *hcl.Block) (*Workspace, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ws := &Workspace{
		Name:      block.Labels[0],
		DeclRange: tfdiags.SourceRangeFromHCL(block.DefRange),
		NameRange: tfdiags.SourceRangeFromHCL(block.LabelRanges[0]),
	}

	if !hclsyntax.ValidIdentifier(ws.Name) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid name for workspace block",
			Detail:   fmt.Sprintf("The name %q is not a valid name for a workspace block. Must start with a letter, followed by zero or more letters, digits, and underscores.", ws.Name),
			Subject:  block.LabelRanges[0].Ptr(),
		})
	}

	content, hclDiags := block.Body.Content(workspaceSchema)
	diags = diags.Append(hclDiags)

	if attr, ok := content.Attributes["for_each"]; ok {
		ws.ForEach = attr.Expr
	}

	if attr, ok := content.Attributes["variables"]; ok {
		ws.Variables = attr.Expr
	}

	if attr, ok := content.Attributes["config"]; ok {
		ws.ConfigSource = attr.Expr
	}

	if attr, ok := content.Attributes["remote"]; ok {
		ws.Remote = attr.Expr
	}

	for _, block := range content.Blocks {
		switch block.Type {
		case "state_storage":
			if ws.StateStorage != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate state_storage block",
					Detail:   fmt.Sprintf("A workspace configuration block may contain at most one state_storage block. State storage was already configured at %s.", ws.StateStorage.DeclRange.StartString()),
					Subject:  block.TypeRange.Ptr(),
				})
				continue
			}

			ss, moreDiags := decodeStateStorageBlock(block)
			diags = diags.Append(moreDiags)
			ws.StateStorage = ss
		default:
			// There are no other block types in our schema
			panic(fmt.Sprintf("unexpected nested block type %q", block.Type))
		}
	}

	return ws, diags
}

// StateStorage represents a state_storage block inside a workspace
// configuration.
type StateStorage struct {
	// TypeName is the name of the storage type, as given in the label of
	// the state_storage block.
	//
	// It is guaranteed to be a valid HCL identifier.
	TypeName string

	// Config is the unevaluated body of the state_storage block, whose
	// content should eventually be evaluated using the configuration schema
	// for the selected storage type.
	Config hcl.Body

	// DeclRange is the source range of the block header of this block,
	// for use in diagnostic messages. TypeNameRange is the range of the
	// TypeName string specifically.
	DeclRange, TypeNameRange tfdiags.SourceRange
}

func decodeStateStorageBlock(block *hcl.Block) (*StateStorage, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ss := &StateStorage{
		TypeName:      block.Labels[0],
		Config:        block.Body,
		DeclRange:     tfdiags.SourceRangeFromHCL(block.DefRange),
		TypeNameRange: tfdiags.SourceRangeFromHCL(block.LabelRanges[0]),
	}

	if !hclsyntax.ValidIdentifier(ss.TypeName) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid state storage type",
			Detail:   fmt.Sprintf("The name %q is not a valid state storage type. Must start with a letter, followed by zero or more letters, digits, and underscores.", ss.TypeName),
			Subject:  block.LabelRanges[0].Ptr(),
		})
	}

	return ss, diags
}

var workspaceSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "for_each"},
		{Name: "variables"},
		{Name: "config"},
		{Name: "remote"},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "state_storage", LabelNames: []string{"type"}},
	},
}
