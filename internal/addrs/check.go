package addrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Check is the address of a check rule within a checkable object.
//
// This represents the check rule globally within a configuration, and is used
// during graph evaluation to identify a condition result object to update with
// the result of check rule evaluation.
//
// The check address is not distinct from resource traversals, and check rule
// values are not intended to be available to the language, so the address is
// not Referenceable.
//
// Note also that the check address is only relevant within the scope of a run,
// as reordering check blocks between runs will result in their addresses
// changing.
type Check struct {
	Container Checkable
	Type      CheckType
	Index     int
}

func (c Check) String() string {
	container := c.Container.String()
	switch c.Type {
	case ResourcePrecondition:
		return fmt.Sprintf("%s.preconditions[%d]", container, c.Index)
	case ResourcePostcondition:
		return fmt.Sprintf("%s.postconditions[%d]", container, c.Index)
	case OutputPrecondition:
		return fmt.Sprintf("%s.preconditions[%d]", container, c.Index)
	default:
		// This should not happen
		return fmt.Sprintf("%s.conditions[%d]", container, c.Index)
	}
}

// Checkable is an interface implemented by all address types that can contain
// condition blocks.
type Checkable interface {
	checkableSigil()

	// Check returns the address of an individual check rule of a specified
	// type and index within this checkable container.
	Check(CheckType, int) Check
	String() string
}

var (
	_ Checkable = AbsResourceInstance{}
	_ Checkable = AbsOutputValue{}
)

type checkable struct {
}

func (c checkable) checkableSigil() {
}

// CheckType describes the category of check.
//go:generate go run golang.org/x/tools/cmd/stringer -type=CheckType check.go
type CheckType int

const (
	InvalidCondition      CheckType = 0
	ResourcePrecondition  CheckType = 1
	ResourcePostcondition CheckType = 2
	OutputPrecondition    CheckType = 3
)

// Description returns a human-readable description of the check type. This is
// presented in the user interface through a diagnostic summary.
func (c CheckType) Description() string {
	switch c {
	case ResourcePrecondition:
		return "Resource precondition"
	case ResourcePostcondition:
		return "Resource postcondition"
	case OutputPrecondition:
		return "Module output value precondition"
	default:
		// This should not happen
		return "Condition"
	}
}

func ParseCheckableStr(raw string) (Checkable, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(raw), "", hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return nil, diags
	}

	path, remain, moreDiags := parseModuleInstancePrefix(traversal)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	if len(remain) < 2 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address operator",
			Detail:   "A checkable object address must have at least two more segments after the module path.",
			Subject:  remain.SourceRange().Ptr(),
		})
		return nil, diags
	}

	switch first := remain[0].(type) {
	case hcl.TraverseRoot:
		switch first.Name {
		case "output":
			if len(remain) > 2 {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid address operator",
					Detail:   "The address for an output value must include only one more attribute after the 'output' keyword, specifying the output value name.",
					Subject:  remain.SourceRange().Ptr(),
				})
				return nil, diags
			}
			nameStep, ok := remain[1].(hcl.TraverseAttr)
			if !ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid address operator",
					Detail:   "The address for an output value must specify the output value name using attribute syntax.",
					Subject:  remain[1].SourceRange().Ptr(),
				})
				return nil, diags
			}
			return AbsOutputValue{
				Module: path,
				OutputValue: OutputValue{
					Name: nameStep.Name,
				},
			}, diags

		case "check":
			// This is reserved to allow us to potentially have top-level "check"
			// blocks later, which serve as general assertions not attached
			// to any particular other object.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid address operator",
				Detail:   "The name 'check' in this context is reserved for a future Terraform language feature.",
				Subject:  first.SourceRange().Ptr(),
			})
			return nil, diags

		default:
			riAddr, moreDiags := parseResourceInstanceUnderModule(path, remain)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return nil, diags
			}
			return riAddr, diags
		}

	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid address operator",
			Detail:   "A checkable object address must have an attribute after the module path indicating the checkable object type.",
			Subject:  remain.SourceRange().Ptr(),
		})
		return nil, diags
	}
}
