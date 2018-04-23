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

func (pc AbsProviderConfig) String() string {
	if len(pc.Module) == 0 {
		return pc.ProviderConfig.String()
	}
	return fmt.Sprintf("%s.%s", pc.Module.String(), pc.ProviderConfig.String())
}
