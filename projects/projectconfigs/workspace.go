package projectconfigs

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"

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

	// ConfigSource and StateStorage are set for local-operations-only
	// workspaces and reflect the "config" argument and the "state_storage"
	// block respectively. Both are nil for remote workspaces.
	Config       hcl.Expression
	StateStorage *StateStorage

	// Remote is the expression given in the "remote" argument for a remote
	// workspace, or nil for local workspaces.
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
	// for use in diagnostic messages. NameRange is the range of the
	// Name string specifically.
	DeclRange, NameRange tfdiags.SourceRange
}

func decodeStateStorageBlock(block *hcl.Block) (*StateStorage, tfdiags.Diagnostics) {
	return nil, nil
}
