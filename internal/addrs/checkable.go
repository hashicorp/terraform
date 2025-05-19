// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Checkable is an interface implemented by all address types that can contain
// condition blocks.
type Checkable interface {
	UniqueKeyer

	checkableSigil()

	// CheckRule returns the address of an individual check rule of a specified
	// type and index within this checkable container.
	CheckRule(CheckRuleType, int) CheckRule

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
)

// CheckableKind describes the different kinds of checkable objects.
type CheckableKind rune

//go:generate go tool golang.org/x/tools/cmd/stringer -type=CheckableKind checkable.go

const (
	CheckableKindInvalid   CheckableKind = 0
	CheckableResource      CheckableKind = 'R'
	CheckableOutputValue   CheckableKind = 'O'
	CheckableCheck         CheckableKind = 'C'
	CheckableInputVariable CheckableKind = 'I'
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

	configCheckableSigil()

	CheckableKind() CheckableKind
	String() string
}

var (
	_ ConfigCheckable = ConfigResource{}
	_ ConfigCheckable = ConfigOutputValue{}
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

	path, remain, diags := parseModuleInstancePrefix(traversal, false)
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

	getCheckableName := func(keyword string, descriptor string) (string, tfdiags.Diagnostics) {
		var diags tfdiags.Diagnostics
		var name string

		if len(remain) != 2 {
			diags = diags.Append(hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid checkable address",
				Detail:   fmt.Sprintf("%s address must have only one attribute part after the keyword '%s', giving the name of the %s.", cases.Title(language.English, cases.NoLower).String(keyword), keyword, descriptor),
				Subject:  remain.SourceRange().Ptr(),
			})
		}

		if remain.RootName() != keyword {
			diags = diags.Append(hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid checkable address",
				Detail:   fmt.Sprintf("%s address must follow the module address with the keyword '%s'.", cases.Title(language.English, cases.NoLower).String(keyword), keyword),
				Subject:  remain.SourceRange().Ptr(),
			})
		}
		if step, ok := remain[1].(hcl.TraverseAttr); !ok {
			diags = diags.Append(hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid checkable address",
				Detail:   fmt.Sprintf("%s address must have only one attribute part after the keyword '%s', giving the name of the %s.", cases.Title(language.English, cases.NoLower).String(keyword), keyword, descriptor),
				Subject:  remain.SourceRange().Ptr(),
			})
		} else {
			name = step.Name
		}

		return name, diags
	}

	// We use "kind" to disambiguate here because unfortunately we've
	// historically never reserved "output" as a possible resource type name
	// and so it is in principle possible -- albeit unlikely -- that there
	// might be a resource whose type is literally "output".
	switch kind {
	case CheckableResource:
		riAddr, moreDiags := parseResourceInstanceUnderModule(path, false, remain)
		diags = diags.Append(moreDiags)
		if diags.HasErrors() {
			return nil, diags
		}
		return riAddr, diags

	case CheckableOutputValue:
		name, nameDiags := getCheckableName("output", "output value")
		diags = diags.Append(nameDiags)
		if diags.HasErrors() {
			return nil, diags
		}
		return OutputValue{Name: name}.Absolute(path), diags

	case CheckableCheck:
		name, nameDiags := getCheckableName("check", "check block")
		diags = diags.Append(nameDiags)
		if diags.HasErrors() {
			return nil, diags
		}
		return Check{Name: name}.Absolute(path), diags

	case CheckableInputVariable:
		name, nameDiags := getCheckableName("var", "variable value")
		diags = diags.Append(nameDiags)
		if diags.HasErrors() {
			return nil, diags
		}
		return InputVariable{Name: name}.Absolute(path), diags

	default:
		panic(fmt.Sprintf("unsupported CheckableKind %s", kind))
	}
}
