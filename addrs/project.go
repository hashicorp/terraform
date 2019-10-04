package addrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/tfdiags"
)

// ProjectWorkspaceConfig refers to a workspace configuration block within
// the current project.
//
// Each configuration block produces zero or more workspaces, whose references
// are represented by ProjectWorkspace.
type ProjectWorkspaceConfig struct {
	Name string
}

// Instance returns the address of a single workspace represented by this
// configuration block.
func (wc ProjectWorkspaceConfig) Instance(key InstanceKey) ProjectWorkspace {
	return ProjectWorkspace{Name: wc.Name, Key: key}
}

func (wc ProjectWorkspaceConfig) String() string {
	return "workspace." + wc.Name
}

// ProjectWorkspace refers to single workspace within the current project.
type ProjectWorkspace struct {
	Name string
	Key  InstanceKey

	projectReferenceable
}

func (w ProjectWorkspace) String() string {
	switch key := w.Key.(type) {
	case nil:
		return "workspace." + w.Name
	case StringKey:
		return fmt.Sprintf("workspace.%s.%s", w.Name, key)
	default:
		// No other key types are valid for project workspaces, but we'll
		// tolerate this anyway for robustness.
		return fmt.Sprintf("workspace.%s%s", w.Name, key.String())
	}
}

// ProjectUpstreamWorkspaceConfig refers to an "upstream" configuration block
// within the current project.
//
// Each configuration block produces zero or more upstream workspaces, whose
// references are represented by ProjectUpstreamWorkspace.
type ProjectUpstreamWorkspaceConfig struct {
	Name string
}

// Instance returns the address of a single workspace represented by this
// configuration block.
func (wc ProjectUpstreamWorkspaceConfig) Instance(key InstanceKey) ProjectUpstreamWorkspace {
	return ProjectUpstreamWorkspace{Name: wc.Name, Key: key}
}

func (wc ProjectUpstreamWorkspaceConfig) String() string {
	return "upstream." + wc.Name
}

// ProjectUpstreamWorkspace refers to a workspace in some other project whose
// outputs are being imported into the current project.
type ProjectUpstreamWorkspace struct {
	Name string
	Key  InstanceKey

	projectReferenceable
}

func (w ProjectUpstreamWorkspace) String() string {
	switch key := w.Key.(type) {
	case nil:
		return "upstream." + w.Name
	case StringKey:
		return fmt.Sprintf("upstream.%s.%s", w.Name, key)
	default:
		// No other key types are valid for project workspaces, but we'll
		// tolerate this anyway for robustness.
		return fmt.Sprintf("upstream.%s%s", w.Name, key.String())
	}
}

// ProjectContextValue refers to a named context value within a project
// configuration. This is similar to InputVariable, but within the definition
// of a project rather than within a module.
type ProjectContextValue struct {
	Name string

	projectReferenceable
}

func (v ProjectContextValue) String() string {
	return "context." + v.Name
}

// ProjectReferenceable is an interface implemented by all address types that
// can appear as references in project definition expressions.
type ProjectReferenceable interface {
	// All implementations of this interface must be covered by the type switch
	// in projectlang.Scope.buildEvalContext.
	projectReferenceableSigil()

	// String produces a string representation of the address that could be
	// parsed as a HCL traversal and passed to ParseProjectConfigRef to produce
	// an equivalent result.
	String() string
}

type projectReferenceable struct {
}

func (r projectReferenceable) projectReferenceableSigil() {
}

// ProjectConfigReference describes a reference to an address with source
// location information, within the project configuration context.
type ProjectConfigReference struct {
	Subject     ProjectReferenceable
	SourceRange tfdiags.SourceRange
	Remaining   hcl.Traversal
}

// ParseProjectConfigRef attempts to extract a referencable address from the
// prefix of the given traversal, which must be an absolute traversal or this
// function will panic.
//
// This function is like ParseRef, but is for project-level configuration
// instead of module-level configuration.
//
// If no error diagnostics are returned, the returned reference includes the
// address that was extracted, the source range it was extracted from, and any
// remaining relative traversal that was not consumed as part of the
// reference.
//
// If error diagnostics are returned then the Reference value is invalid and
// must not be used.
func ParseProjectConfigRef(traversal hcl.Traversal) (*ProjectConfigReference, tfdiags.Diagnostics) {
	ref, diags := parseProjectConfigRef(traversal)

	// Normalize a little to make life easier for callers.
	if ref != nil {
		if len(ref.Remaining) == 0 {
			ref.Remaining = nil
		}
	}

	return ref, diags
}

func parseProjectConfigRef(traversal hcl.Traversal) (*ProjectConfigReference, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	root := traversal.RootName()
	rootRange := traversal[0].SourceRange()

	switch root {

	case "each":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &ProjectConfigReference{
			Subject:     ForEachAttr{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "local":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &ProjectConfigReference{
			Subject:     LocalValue{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "context":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &ProjectConfigReference{
			Subject:     ProjectContextValue{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "workspace":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &ProjectConfigReference{
			Subject:     ProjectWorkspace{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	case "upstream":
		name, rng, remain, diags := parseSingleAttrRef(traversal)
		return &ProjectConfigReference{
			Subject:     ProjectUpstreamWorkspace{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(rng),
			Remaining:   remain,
		}, diags

	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   fmt.Sprintf("There is no object %q in the project-level configuration language.", root),
			Subject:  rootRange.Ptr(),
			Context:  traversal.SourceRange().Ptr(),
		})
		return nil, diags
	}
}
