package addrs

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ProviderConfig is an interface type whose dynamic type can be either
// LocalProviderConfig or AbsProviderConfig, in order to represent situations
// where a value might either be module-local or absolute but the decision
// cannot be made until runtime.
//
// Where possible, use either LocalProviderConfig or AbsProviderConfig directly
// instead, to make intent more clear. ProviderConfig can be used only in
// situations where the recipient of the value has some out-of-band way to
// determine a "current module" to use if the value turns out to be
// a LocalProviderConfig.
//
// Recipients of non-nil ProviderConfig values that actually need
// AbsProviderConfig values should call ResolveAbsProviderAddr on the
// *configs.Config value representing the root module configuration, which
// handles the translation from local to fully-qualified using mapping tables
// defined in the configuration.
//
// Recipients of a ProviderConfig value can assume it can contain only a
// LocalProviderConfig value, an AbsProviderConfigValue, or nil to represent
// the absense of a provider config in situations where that is meaningful.
type ProviderConfig interface {
	providerConfig()
}

// LocalProviderConfig is the address of a provider configuration from the
// perspective of references in a particular module.
//
// Finding the corresponding AbsProviderConfig will require looking up the
// LocalName in the providers table in the module's configuration; there is
// no syntax-only translation between these types.
type LocalProviderConfig struct {
	LocalName string

	// If not empty, Alias identifies which non-default (aliased) provider
	// configuration this address refers to.
	Alias string
}

var _ ProviderConfig = LocalProviderConfig{}

// NewDefaultLocalProviderConfig returns the address of the default (un-aliased)
// configuration for the provider with the given local type name.
func NewDefaultLocalProviderConfig(LocalNameName string) LocalProviderConfig {
	return LocalProviderConfig{
		LocalName: LocalNameName,
	}
}

// providerConfig Implements addrs.ProviderConfig.
func (pc LocalProviderConfig) providerConfig() {}

func (pc LocalProviderConfig) String() string {
	if pc.LocalName == "" {
		// Should never happen; always indicates a bug
		return "provider.<invalid>"
	}

	if pc.Alias != "" {
		return fmt.Sprintf("provider.%s.%s", pc.LocalName, pc.Alias)
	}

	return "provider." + pc.LocalName
}

// StringCompact is an alternative to String that returns the form that can
// be parsed by ParseProviderConfigCompact, without the "provider." prefix.
func (pc LocalProviderConfig) StringCompact() string {
	if pc.Alias != "" {
		return fmt.Sprintf("%s.%s", pc.LocalName, pc.Alias)
	}
	return pc.LocalName
}

// AbsProviderConfig is the absolute address of a provider configuration
// within a particular module instance.
type AbsProviderConfig struct {
	Module   Module
	Provider Provider
	Alias    string
}

var _ ProviderConfig = AbsProviderConfig{}

// ParseAbsProviderConfig parses the given traversal as an absolute provider
// address. The following are examples of traversals that can be successfully
// parsed as absolute provider configuration addresses:
//
//     provider["registry.terraform.io/hashicorp/aws"]
//     provider["registry.terraform.io/hashicorp/aws"].foo
//     module.bar.provider["registry.terraform.io/hashicorp/aws"]
//     module.bar.module.baz.provider["registry.terraform.io/hashicorp/aws"].foo
//
// This type of address is used, for example, to record the relationships
// between resources and provider configurations in the state structure.
// This type of address is not generally used in the UI, except in error
// messages that refer to provider configurations.
func ParseAbsProviderConfig(traversal hcl.Traversal) (AbsProviderConfig, tfdiags.Diagnostics) {
	modInst, remain, diags := parseModuleInstancePrefix(traversal)
	var ret AbsProviderConfig

	// Providers cannot resolve within module instances, so verify that there
	// are no instance keys in the module path before converting to a Module.
	for _, step := range modInst {
		if step.InstanceKey != NoKey {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider configuration address",
				Detail:   "Provider address cannot contain module indexes",
				Subject:  remain.SourceRange().Ptr(),
			})
			return ret, diags
		}
	}
	ret.Module = modInst.Module()

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

	if tt, ok := remain[1].(hcl.TraverseIndex); ok {
		if !tt.Key.Type().Equals(cty.String) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider configuration address",
				Detail:   "The prefix \"provider.\" must be followed by a provider type name.",
				Subject:  remain[1].SourceRange().Ptr(),
			})
			return ret, diags
		}
		p, sourceDiags := ParseProviderSourceString(tt.Key.AsString())
		ret.Provider = p
		if sourceDiags.HasErrors() {
			diags = diags.Append(sourceDiags)
			return ret, diags
		}
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
			ret.Alias = tt.Name
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

// ParseAbsProviderConfigStr is a helper wrapper around ParseAbsProviderConfig
// that takes a string and parses it with the HCL native syntax traversal parser
// before interpreting it.
//
// This should be used only in specialized situations since it will cause the
// created references to not have any meaningful source location information.
// If a reference string is coming from a source that should be identified in
// error messages then the caller should instead parse it directly using a
// suitable function from the HCL API and pass the traversal itself to
// ParseAbsProviderConfig.
//
// Error diagnostics are returned if either the parsing fails or the analysis
// of the traversal fails. There is no way for the caller to distinguish the
// two kinds of diagnostics programmatically. If error diagnostics are returned
// the returned address is invalid.
func ParseAbsProviderConfigStr(str string) (AbsProviderConfig, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return AbsProviderConfig{}, diags
	}
	addr, addrDiags := ParseAbsProviderConfig(traversal)
	diags = diags.Append(addrDiags)
	return addr, diags
}

func ParseLegacyAbsProviderConfigStr(str string) (AbsProviderConfig, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return AbsProviderConfig{}, diags
	}

	addr, addrDiags := ParseLegacyAbsProviderConfig(traversal)
	diags = diags.Append(addrDiags)
	return addr, diags
}

// ParseLegacyAbsProviderConfig parses the given traversal as an absolute
// provider address. The following are examples of traversals that can be
// successfully parsed as legacy absolute provider configuration addresses:
//
//     provider.aws
//     provider.aws.foo
//     module.bar.provider.aws
//     module.bar.module.baz.provider.aws.foo
//
// This type of address is used in legacy state and may appear in state v4 if
// the provider config addresses have not been normalized to include provider
// FQN.
func ParseLegacyAbsProviderConfig(traversal hcl.Traversal) (AbsProviderConfig, tfdiags.Diagnostics) {
	modInst, remain, diags := parseModuleInstancePrefix(traversal)
	var ret AbsProviderConfig

	// Providers cannot resolve within module instances, so verify that there
	// are no instance keys in the module path before converting to a Module.
	for _, step := range modInst {
		if step.InstanceKey != NoKey {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider configuration address",
				Detail:   "Provider address cannot contain module indexes",
				Subject:  remain.SourceRange().Ptr(),
			})
			return ret, diags
		}
	}
	ret.Module = modInst.Module()

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

	// We always assume legacy-style providers in legacy state ...
	if tt, ok := remain[1].(hcl.TraverseAttr); ok {
		// ... unless it's the builtin "terraform" provider, a special case.
		if tt.Name == "terraform" {
			ret.Provider = NewBuiltInProvider(tt.Name)
		} else {
			ret.Provider = NewLegacyProvider(tt.Name)
		}
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
			ret.Alias = tt.Name
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

// ProviderConfigDefault returns the address of the default provider config of
// the given type inside the recieving module instance.
func (m ModuleInstance) ProviderConfigDefault(provider Provider) AbsProviderConfig {
	return AbsProviderConfig{
		Module:   m.Module(),
		Provider: provider,
	}
}

// ProviderConfigAliased returns the address of an aliased provider config of
// the given type and alias inside the recieving module instance.
func (m ModuleInstance) ProviderConfigAliased(provider Provider, alias string) AbsProviderConfig {
	return AbsProviderConfig{
		Module:   m.Module(),
		Provider: provider,
		Alias:    alias,
	}
}

// providerConfig Implements addrs.ProviderConfig.
func (pc AbsProviderConfig) providerConfig() {}

// Inherited returns an address that the receiving configuration address might
// inherit from in a parent module. The second bool return value indicates if
// such inheritance is possible, and thus whether the returned address is valid.
//
// Inheritance is possible only for default (un-aliased) providers in modules
// other than the root module. Even if a valid address is returned, inheritence
// may not be performed for other reasons, such as if the calling module
// provided explicit provider configurations within the call for this module.
// The ProviderTransformer graph transform in the main terraform module has the
// authoritative logic for provider inheritance, and this method is here mainly
// just for its benefit.
func (pc AbsProviderConfig) Inherited() (AbsProviderConfig, bool) {
	// Can't inherit if we're already in the root.
	if len(pc.Module) == 0 {
		return AbsProviderConfig{}, false
	}

	// Can't inherit if we have an alias.
	if pc.Alias != "" {
		return AbsProviderConfig{}, false
	}

	// Otherwise, we might inherit from a configuration with the same
	// provider type in the parent module instance.
	parentMod := pc.Module.Parent()
	return AbsProviderConfig{
		Module:   parentMod,
		Provider: pc.Provider,
	}, true

}

// LegacyString() returns a legacy-style AbsProviderConfig string and should only be used for legacy state shimming.
func (pc AbsProviderConfig) LegacyString() string {
	if pc.Alias != "" {
		if len(pc.Module) == 0 {
			return fmt.Sprintf("%s.%s.%s", "provider", pc.Provider.LegacyString(), pc.Alias)
		} else {
			return fmt.Sprintf("%s.%s.%s.%s", pc.Module.String(), "provider", pc.Provider.LegacyString(), pc.Alias)
		}
	}
	if len(pc.Module) == 0 {
		return fmt.Sprintf("%s.%s", "provider", pc.Provider.LegacyString())
	}
	return fmt.Sprintf("%s.%s.%s", pc.Module.String(), "provider", pc.Provider.LegacyString())
}

// String() returns a string representation of an AbsProviderConfig in the following format:
//
// 	provider["example.com/namespace/name"]
// 	provider["example.com/namespace/name"].alias
// 	module.module-name.provider["example.com/namespace/name"]
// 	module.module-name.provider["example.com/namespace/name"].alias
func (pc AbsProviderConfig) String() string {
	var parts []string
	if len(pc.Module) > 0 {
		parts = append(parts, pc.Module.String())
	}

	parts = append(parts, fmt.Sprintf("provider[%q]", pc.Provider))

	if pc.Alias != "" {
		parts = append(parts, pc.Alias)
	}

	return strings.Join(parts, ".")
}
