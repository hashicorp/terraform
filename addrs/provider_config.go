package addrs

import (
	"fmt"

	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/hcl2/hcl"
)

// ProviderConfig is the address of a provider configuration.
type ProviderConfig struct {
	Type string

	// If not empty, Alias identifies which non-default (aliased) provider
	// configuration this address refers to.
	Alias string
}

// NewDefaultProviderConfig returns the address of the default (un-aliased)
// configuration for the provider with the given type name.
func NewDefaultProviderConfig(typeName string) ProviderConfig {
	return ProviderConfig{
		Type: typeName,
	}
}

// ParseProviderConfigCompact parses the given absolute traversal as a relative
// provider address in compact form. The following are examples of traversals
// that can be successfully parsed as compact relative provider configuration
// addresses:
//
//     aws
//     aws.foo
//
// This function will panic if given a relative traversal.
//
// If the returned diagnostics contains errors then the result value is invalid
// and must not be used.
func ParseProviderConfigCompact(traversal hcl.Traversal) (ProviderConfig, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := ProviderConfig{
		Type: traversal.RootName(),
	}

	if len(traversal) < 2 {
		// Just a type name, then.
		return ret, diags
	}

	aliasStep := traversal[1]
	switch ts := aliasStep.(type) {
	case hcl.TraverseAttr:
		ret.Alias = ts.Name
		return ret, diags
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider configuration address",
			Detail:   "The provider type name must either stand alone or be followed by an alias name separated with a dot.",
			Subject:  aliasStep.SourceRange().Ptr(),
		})
	}

	if len(traversal) > 2 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider configuration address",
			Detail:   "Extraneous extra operators after provider configuration address.",
			Subject:  traversal[2:].SourceRange().Ptr(),
		})
	}

	return ret, diags
}

// Absolute returns an AbsProviderConfig from the receiver and the given module
// instance address.
func (pc ProviderConfig) Absolute(module ModuleInstance) AbsProviderConfig {
	return AbsProviderConfig{
		Module:         module,
		ProviderConfig: pc,
	}
}

func (pc ProviderConfig) String() string {
	if pc.Alias != "" {
		return fmt.Sprintf("provider.%s.%s", pc.Type, pc.Alias)
	}

	return "provider." + pc.Type
}

// StringCompact is an alternative to String that returns the form that can
// be parsed by ParseProviderConfigCompact, without the "provider." prefix.
func (pc ProviderConfig) StringCompact() string {
	if pc.Alias != "" {
		return fmt.Sprintf("%s.%s", pc.Type, pc.Alias)
	}
	return pc.Type
}

// AbsProviderConfig is the absolute address of a provider configuration
// within a particular module instance.
type AbsProviderConfig struct {
	Module         ModuleInstance
	ProviderConfig ProviderConfig
}

// ParseAbsProviderConfig parses the given traversal as an absolute provider
// address. The following are examples of traversals that can be successfully
// parsed as absolute provider configuration addresses:
//
//     provider.aws
//     provider.aws.foo
//     module.bar.provider.aws
//     module.bar.module.baz.provider.aws.foo
//     module.foo[1].provider.aws.foo
//
// This type of address is used, for example, to record the relationships
// between resources and provider configurations in the state structure.
// This type of address is not generally used in the UI, except in error
// messages that refer to provider configurations.
func ParseAbsProviderConfig(traversal hcl.Traversal) (AbsProviderConfig, tfdiags.Diagnostics) {
	modInst, remain, diags := parseModuleInstancePrefix(traversal)
	ret := AbsProviderConfig{
		Module: modInst,
	}
	if len(remain) < 2 || remain.RootName() != "provider" {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider configuration address",
			Detail:   "Provider address must begin with \"provider.\", followed by a provider type name.",
			Subject:  remain.SourceRange().Ptr(),
		})
		return ret, diags
	}
	if len(remain) > 3 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider configuration address",
			Detail:   "Extraneous operators after provider configuration alias.",
			Subject:  hcl.Traversal(remain[3:]).SourceRange().Ptr(),
		})
		return ret, diags
	}

	if tt, ok := remain[1].(hcl.TraverseAttr); ok {
		ret.ProviderConfig.Type = tt.Name
	} else {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider configuration address",
			Detail:   "The prefix \"provider.\" must be followed by a provider type name.",
			Subject:  remain[1].SourceRange().Ptr(),
		})
		return ret, diags
	}

	if len(remain) == 3 {
		if tt, ok := remain[2].(hcl.TraverseAttr); ok {
			ret.ProviderConfig.Alias = tt.Name
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider configuration address",
				Detail:   "Provider type name must be followed by a configuration alias name.",
				Subject:  remain[2].SourceRange().Ptr(),
			})
			return ret, diags
		}
	}

	return ret, diags
}

// ProviderConfigDefault returns the address of the default provider config
// of the given type inside the recieving module instance.
func (m ModuleInstance) ProviderConfigDefault(name string) AbsProviderConfig {
	return AbsProviderConfig{
		Module: m,
		ProviderConfig: ProviderConfig{
			Type: name,
		},
	}
}

// ProviderConfigAliased returns the address of an aliased provider config
// of with given type and alias inside the recieving module instance.
func (m ModuleInstance) ProviderConfigAliased(name, alias string) AbsProviderConfig {
	return AbsProviderConfig{
		Module: m,
		ProviderConfig: ProviderConfig{
			Type:  name,
			Alias: alias,
		},
	}
}

// Inherited returns an address that the receiving configuration address might
// inherit from in a parent module. The second bool return value indicates if
// such inheritance is possible, and thus whether the returned address is valid.
//
// Inheritance is possible only for default (un-aliased) providers in modules
// other than the root module. Even if a valid address is returned, inheritence
// may not be performed for other reasons, such as if the calling module
// provided explicit provider configurations within the call for this module.
// The ProviderTransformer graph transform in the main terraform module has
// the authoritative logic for provider inheritance, and this method is here
// mainly just for its benefit.
func (pc AbsProviderConfig) Inherited() (AbsProviderConfig, bool) {
	// Can't inherit if we're already in the root.
	if len(pc.Module) == 0 {
		return AbsProviderConfig{}, false
	}

	// Can't inherit if we have an alias.
	if pc.ProviderConfig.Alias != "" {
		return AbsProviderConfig{}, false
	}

	// Otherwise, we might inherit from a configuration with the same
	// provider name in the parent module instance.
	parentMod := pc.Module.Parent()
	return pc.ProviderConfig.Absolute(parentMod), true
}

func (pc AbsProviderConfig) String() string {
	if len(pc.Module) == 0 {
		return pc.ProviderConfig.String()
	}
	return fmt.Sprintf("%s.%s", pc.Module.String(), pc.ProviderConfig.String())
}
