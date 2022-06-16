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

func NewCheck(container Checkable, typ CheckType, index int) Check {
	return Check{
		Container: container,
		Type:      typ,
		Index:     index,
	}
}

func (c Check) String() string {
	container := c.Container.String()
	switch c.Type {
	case ResourcePrecondition:
		return fmt.Sprintf("%s.precondition[%d]", container, c.Index)
	case ResourcePostcondition:
		return fmt.Sprintf("%s.postcondition[%d]", container, c.Index)
	case OutputPrecondition:
		return fmt.Sprintf("%s.precondition[%d]", container, c.Index)
	default:
		// This should not happen
		return fmt.Sprintf("%s.condition[%d]", container, c.Index)
	}
}

func (c Check) UniqueKey() UniqueKey {
	return checkKey{
		ContainerKey: c.Container.UniqueKey(),
		Type:         c.Type,
		Index:        c.Index,
	}
}

type checkKey struct {
	ContainerKey UniqueKey
	Type         CheckType
	Index        int
}

func (k checkKey) uniqueKeySigil() {}

// Checkable is an interface implemented by all address types that can contain
// condition blocks.
type Checkable interface {
	UniqueKeyer

	checkableSigil()

	// Check returns the address of an individual check rule of a specified
	// type and index within this checkable container.
	Check(CheckType, int) Check

	// ConfigCheckable returns the address of the configuration construct that
	// this Checkable belongs to.
	//
	// Checkable objects can potentially be dynamically declared during a
	// plan operation using constructs like resource for_each, and so
	// ConfigCheckable gives us a way to talk about the static containers
	// those dynamic objects belong to, in case we wish to group together
	// dynamic checkable objects into their static checkable for reporting
	// purposes.
	ConfigCheckable() ConfigCheckable

	String() string
}

var (
	_ Checkable = AbsResourceInstance{}
	_ Checkable = AbsOutputValue{}
)

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

// ConfigCheckable is an interfaces implemented by address types that represent
// configuration constructs that can have Checkable addresses associated with
// them.
//
// This address type therefore in a sense represents a container for zero or
// more checkable objects all declared by the same configuration construct,
// so that we can talk about these groups of checkable objects before we're
// ready to decide how many checkable objects belong to each one.
type ConfigCheckable interface {
	UniqueKeyer

	configCheckableSigil()

	String() string
}

var (
	_ ConfigCheckable = ConfigResource{}
	_ ConfigCheckable = ConfigOutputValue{}
)

// ParseCheckableStr attempts to parse the given string as a Checkable address.
//
// This should be the opposite of Checkable.String for any Checkable address
// type.
//
// We do not typically expect users to write out checkable addresses as input,
// but we use them as part of some of our wire formats for persisting check
// results between runs.
func ParseCheckableStr(src string) (Checkable, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(src), "", hcl.InitialPos)
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return nil, diags
	}

	path, remain, diags := parseModuleInstancePrefix(traversal)
	if diags.HasErrors() {
		return nil, diags
	}

	if remain.IsRelative() {
		// (relative means that there's either nothing left or what's next isn't an identifier)
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid checkable address",
			Detail:   "Module path must be followed by either a resource instance address or an output value address.",
			Subject:  remain.SourceRange().Ptr(),
		})
		return nil, diags
	}

	switch remain.RootName() {
	case "output":
		if len(remain) != 2 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid checkable address",
				Detail:   "Output address must have only one attribute part after the keyword 'output', giving the name of the output value.",
				Subject:  remain.SourceRange().Ptr(),
			})
			return nil, diags
		}
		if step, ok := remain[1].(hcl.TraverseAttr); !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid checkable address",
				Detail:   "Output address must have only one attribute part after the keyword 'output', giving the name of the output value.",
				Subject:  remain.SourceRange().Ptr(),
			})
			return nil, diags
		} else {
			return OutputValue{Name: step.Name}.Absolute(path), diags
		}
	default:
		riAddr, moreDiags := parseResourceInstanceUnderModule(path, remain)
		diags = diags.Append(moreDiags)
		if diags.HasErrors() {
			return nil, diags
		}
		return riAddr, diags
	}
}
