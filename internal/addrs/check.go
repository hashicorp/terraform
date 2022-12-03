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
// changing. Check is therefore for internal use only and should not be exposed
// in durable artifacts such as state snapshots.
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

// CheckType describes a category of check. We use this only to establish
// uniqueness for Check values, and do not expose this concept of "check types"
// (which is subject to change in future) in any durable artifacts such as
// state snapshots.
//
// (See [CheckableKind] for an enumeration that we _do_ use externally, to
// describe the type of object being checked rather than the type of the check
// itself.)
type CheckType int

//go:generate go run golang.org/x/tools/cmd/stringer -type=CheckType check.go

const (
	InvalidCondition       CheckType = 0
	ResourcePrecondition   CheckType = 1
	ResourcePostcondition  CheckType = 2
	OutputPrecondition     CheckType = 3
	SmokeTestPrecondition  CheckType = 4
	SmokeTestPostcondition CheckType = 5
	SmokeTestDataResource  CheckType = 6
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
	case SmokeTestPrecondition:
		return "Smoke test precondition"
	case SmokeTestPostcondition:
		return "Smoke test postcondition"
	case SmokeTestDataResource:
		return "Smoke test data resource"
	default:
		// This should not happen
		return "Condition"
	}
}

// Checkable is an interface implemented by all address types that can contain
// condition blocks.
type Checkable interface {
	UniqueKeyer
	InModuleInstance

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

	CheckableKind() CheckableKind
	String() string
}

var (
	_ Checkable = AbsResourceInstance{}
	_ Checkable = AbsOutputValue{}
	_ Checkable = AbsSmokeTest{}
)

// CheckableKind describes the different kinds of checkable objects.
type CheckableKind rune

//go:generate go run golang.org/x/tools/cmd/stringer -type=CheckableKind check.go

const (
	CheckableKindInvalid CheckableKind = 0
	CheckableResource    CheckableKind = 'R'
	CheckableOutputValue CheckableKind = 'O'
	CheckableSmokeTest   CheckableKind = 'S'
)

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
	InModule

	configCheckableSigil()

	CheckableKind() CheckableKind
	String() string
}

var (
	_ ConfigCheckable = ConfigResource{}
	_ ConfigCheckable = ConfigOutputValue{}
	_ ConfigCheckable = ConfigSmokeTest{}
)

// ParseCheckableStr attempts to parse the given string as a Checkable address
// of the given kind.
//
// This should be the opposite of Checkable.String for any Checkable address
// type, as long as "kind" is set to the value returned by the address's
// CheckableKind method.
//
// We do not typically expect users to write out checkable addresses as input,
// but we use them as part of some of our wire formats for persisting check
// results between runs.
func ParseCheckableStr(kind CheckableKind, src string) (Checkable, tfdiags.Diagnostics) {
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

	// We use "kind" to disambiguate here because unfortunately we've
	// historically never reserved "output" as a possible resource type name
	// and so it is in principle possible -- albeit unlikely -- that there
	// might be a resource whose type is literally "output".
	switch kind {
	case CheckableResource:
		riAddr, moreDiags := parseResourceInstanceUnderModule(path, remain)
		diags = diags.Append(moreDiags)
		if diags.HasErrors() {
			return nil, diags
		}
		return riAddr, diags

	case CheckableOutputValue:
		if len(remain) != 2 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid checkable address",
				Detail:   "Output address must have only one attribute part after the keyword 'output', giving the name of the output value.",
				Subject:  remain.SourceRange().Ptr(),
			})
			return nil, diags
		}
		if remain.RootName() != "output" {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid checkable address",
				Detail:   "Output address must follow the module address with the keyword 'output'.",
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

	case CheckableSmokeTest:
		if len(remain) != 2 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid checkable address",
				Detail:   "Smoke test address must have only one attribute part after the keyword 'smoke_test', giving the name of the smoke test.",
				Subject:  remain.SourceRange().Ptr(),
			})
			return nil, diags
		}
		if remain.RootName() != "smoke_test" {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid checkable address",
				Detail:   "Smoke test address must follow the module address with the keyword 'smoke_test'.",
				Subject:  remain.SourceRange().Ptr(),
			})
			return nil, diags
		}
		if step, ok := remain[1].(hcl.TraverseAttr); !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid checkable address",
				Detail:   "Smoke test address must have only one attribute part after the keyword 'smoke_test', giving the name of the smoke test.",
				Subject:  remain.SourceRange().Ptr(),
			})
			return nil, diags
		} else {
			return SmokeTest{Name: step.Name}.Absolute(path), diags
		}

	default:
		panic(fmt.Sprintf("unsupported CheckableKind %s", kind))
	}
}
