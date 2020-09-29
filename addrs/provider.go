package addrs

import (
	"fmt"
	"strings"

	"golang.org/x/net/idna"

	"github.com/hashicorp/hcl/v2"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/tfdiags"
)

// Provider encapsulates a single provider type. In the future this will be
// extended to include additional fields including Namespace and SourceHost
type Provider struct {
	Type      string
	Namespace string
	Hostname  svchost.Hostname
}

// DefaultRegistryHost is the hostname used for provider addresses that do
// not have an explicit hostname.
const DefaultRegistryHost = svchost.Hostname("registry.terraform.io")

// BuiltInProviderHost is the pseudo-hostname used for the "built-in" provider
// namespace. Built-in provider addresses must also have their namespace set
// to BuiltInProviderNamespace in order to be considered as built-in.
const BuiltInProviderHost = svchost.Hostname("terraform.io")

// BuiltInProviderNamespace is the provider namespace used for "built-in"
// providers. Built-in provider addresses must also have their hostname
// set to BuiltInProviderHost in order to be considered as built-in.
//
// The this namespace is literally named "builtin", in the hope that users
// who see FQNs containing this will be able to infer the way in which they are
// special, even if they haven't encountered the concept formally yet.
const BuiltInProviderNamespace = "builtin"

// LegacyProviderNamespace is the special string used in the Namespace field
// of type Provider to mark a legacy provider address. This special namespace
// value would normally be invalid, and can be used only when the hostname is
// DefaultRegistryHost because that host owns the mapping from legacy name to
// FQN.
const LegacyProviderNamespace = "-"

// String returns an FQN string, indended for use in machine-readable output.
func (pt Provider) String() string {
	if pt.IsZero() {
		panic("called String on zero-value addrs.Provider")
	}
	return pt.Hostname.ForDisplay() + "/" + pt.Namespace + "/" + pt.Type
}

// ForDisplay returns a user-friendly FQN string, simplified for readability. If
// the provider is using the default hostname, the hostname is omitted.
func (pt Provider) ForDisplay() string {
	if pt.IsZero() {
		panic("called ForDisplay on zero-value addrs.Provider")
	}

	if pt.Hostname == DefaultRegistryHost {
		return pt.Namespace + "/" + pt.Type
	}
	return pt.Hostname.ForDisplay() + "/" + pt.Namespace + "/" + pt.Type
}

// NewProvider constructs a provider address from its parts, and normalizes
// the namespace and type parts to lowercase using unicode case folding rules
// so that resulting addrs.Provider values can be compared using standard
// Go equality rules (==).
//
// The hostname is given as a svchost.Hostname, which is required by the
// contract of that type to have already been normalized for equality testing.
//
// This function will panic if the given namespace or type name are not valid.
// When accepting namespace or type values from outside the program, use
// ParseProviderPart first to check that the given value is valid.
func NewProvider(hostname svchost.Hostname, namespace, typeName string) Provider {
	if namespace == LegacyProviderNamespace {
		// Legacy provider addresses must always be created via
		// NewLegacyProvider so that we can use static analysis to find
		// codepaths still working with those.
		panic("attempt to create legacy provider address using NewProvider; use NewLegacyProvider instead")
	}

	return Provider{
		Type:      MustParseProviderPart(typeName),
		Namespace: MustParseProviderPart(namespace),
		Hostname:  hostname,
	}
}

// ImpliedProviderForUnqualifiedType represents the rules for inferring what
// provider FQN a user intended when only a naked type name is available.
//
// For all except the type name "terraform" this returns a so-called "default"
// provider, which is under the registry.terraform.io/hashicorp/ namespace.
//
// As a special case, the string "terraform" maps to
// "terraform.io/builtin/terraform" because that is the more likely user
// intent than the now-unmaintained "registry.terraform.io/hashicorp/terraform"
// which remains only for compatibility with older Terraform versions.
func ImpliedProviderForUnqualifiedType(typeName string) Provider {
	switch typeName {
	case "terraform":
		// Note for future maintainers: any additional strings we add here
		// as implied to be builtin must never also be use as provider names
		// in the registry.terraform.io/hashicorp/... namespace, because
		// otherwise older versions of Terraform could implicitly select
		// the registry name instead of the internal one.
		return NewBuiltInProvider(typeName)
	default:
		return NewDefaultProvider(typeName)
	}
}

// NewDefaultProvider returns the default address of a HashiCorp-maintained,
// Registry-hosted provider.
func NewDefaultProvider(name string) Provider {
	return Provider{
		Type:      MustParseProviderPart(name),
		Namespace: "hashicorp",
		Hostname:  DefaultRegistryHost,
	}
}

// NewBuiltInProvider returns the address of a "built-in" provider. See
// the docs for Provider.IsBuiltIn for more information.
func NewBuiltInProvider(name string) Provider {
	return Provider{
		Type:      MustParseProviderPart(name),
		Namespace: BuiltInProviderNamespace,
		Hostname:  BuiltInProviderHost,
	}
}

// NewLegacyProvider returns a mock address for a provider.
// This will be removed when ProviderType is fully integrated.
func NewLegacyProvider(name string) Provider {
	return Provider{
		// We intentionally don't normalize and validate the legacy names,
		// because existing code expects legacy provider names to pass through
		// verbatim, even if not compliant with our new naming rules.
		Type:      name,
		Namespace: LegacyProviderNamespace,
		Hostname:  DefaultRegistryHost,
	}
}

// LegacyString returns the provider type, which is frequently used
// interchangeably with provider name. This function can and should be removed
// when provider type is fully integrated. As a safeguard for future
// refactoring, this function panics if the Provider is not a legacy provider.
func (pt Provider) LegacyString() string {
	if pt.IsZero() {
		panic("called LegacyString on zero-value addrs.Provider")
	}
	if pt.Namespace != LegacyProviderNamespace && pt.Namespace != BuiltInProviderNamespace {
		panic(pt.String() + " cannot be represented as a legacy string")
	}
	return pt.Type
}

// IsZero returns true if the receiver is the zero value of addrs.Provider.
//
// The zero value is not a valid addrs.Provider and calling other methods on
// such a value is likely to either panic or otherwise misbehave.
func (pt Provider) IsZero() bool {
	return pt == Provider{}
}

// IsBuiltIn returns true if the receiver is the address of a "built-in"
// provider. That is, a provider under terraform.io/builtin/ which is
// included as part of the Terraform binary itself rather than one to be
// installed from elsewhere.
//
// These are ignored by the provider installer because they are assumed to
// already be available without any further installation.
func (pt Provider) IsBuiltIn() bool {
	return pt.Hostname == BuiltInProviderHost && pt.Namespace == BuiltInProviderNamespace
}

// LessThan returns true if the receiver should sort before the other given
// address in an ordered list of provider addresses.
//
// This ordering is an arbitrary one just to allow deterministic results from
// functions that would otherwise have no natural ordering. It's subject
// to change in future.
func (pt Provider) LessThan(other Provider) bool {
	switch {
	case pt.Hostname != other.Hostname:
		return pt.Hostname < other.Hostname
	case pt.Namespace != other.Namespace:
		return pt.Namespace < other.Namespace
	default:
		return pt.Type < other.Type
	}
}

// IsLegacy returns true if the provider is a legacy-style provider
func (pt Provider) IsLegacy() bool {
	if pt.IsZero() {
		panic("called IsLegacy() on zero-value addrs.Provider")
	}

	return pt.Hostname == DefaultRegistryHost && pt.Namespace == LegacyProviderNamespace

}

// IsDefault returns true if the provider is a default hashicorp provider
func (pt Provider) IsDefault() bool {
	if pt.IsZero() {
		panic("called IsDefault() on zero-value addrs.Provider")
	}

	return pt.Hostname == DefaultRegistryHost && pt.Namespace == "hashicorp"
}

// Equals returns true if the receiver and other provider have the same attributes.
func (pt Provider) Equals(other Provider) bool {
	return pt == other
}

// ParseProviderSourceString parses the source attribute and returns a provider.
// This is intended primarily to parse the FQN-like strings returned by
// terraform-config-inspect.
//
// The following are valid source string formats:
// 		name
// 		namespace/name
// 		hostname/namespace/name
func ParseProviderSourceString(str string) (Provider, tfdiags.Diagnostics) {
	var ret Provider
	var diags tfdiags.Diagnostics

	// split the source string into individual components
	parts := strings.Split(str, "/")
	if len(parts) == 0 || len(parts) > 3 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider source string",
			Detail:   `The "source" attribute must be in the format "[hostname/][namespace/]name"`,
		})
		return ret, diags
	}

	// check for an invalid empty string in any part
	for i := range parts {
		if parts[i] == "" {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider source string",
				Detail:   `The "source" attribute must be in the format "[hostname/][namespace/]name"`,
			})
			return ret, diags
		}
	}

	// check the 'name' portion, which is always the last part
	givenName := parts[len(parts)-1]
	name, err := ParseProviderPart(givenName)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider type",
			Detail:   fmt.Sprintf(`Invalid provider type %q in source %q: %s"`, givenName, str, err),
		})
		return ret, diags
	}
	ret.Type = name
	ret.Hostname = DefaultRegistryHost

	if len(parts) == 1 {
		return NewDefaultProvider(parts[0]), diags
	}

	if len(parts) >= 2 {
		// the namespace is always the second-to-last part
		givenNamespace := parts[len(parts)-2]
		if givenNamespace == LegacyProviderNamespace {
			// For now we're tolerating legacy provider addresses until we've
			// finished updating the rest of the codebase to no longer use them,
			// or else we'd get errors round-tripping through legacy subsystems.
			ret.Namespace = LegacyProviderNamespace
		} else {
			namespace, err := ParseProviderPart(givenNamespace)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid provider namespace",
					Detail:   fmt.Sprintf(`Invalid provider namespace %q in source %q: %s"`, namespace, str, err),
				})
				return Provider{}, diags
			}
			ret.Namespace = namespace
		}
	}

	// Final Case: 3 parts
	if len(parts) == 3 {
		// the namespace is always the first part in a three-part source string
		hn, err := svchost.ForComparison(parts[0])
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider source hostname",
				Detail:   fmt.Sprintf(`Invalid provider source hostname namespace %q in source %q: %s"`, hn, str, err),
			})
			return Provider{}, diags
		}
		ret.Hostname = hn
	}

	if ret.Namespace == LegacyProviderNamespace && ret.Hostname != DefaultRegistryHost {
		// Legacy provider addresses must always be on the default registry
		// host, because the default registry host decides what actual FQN
		// each one maps to.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider namespace",
			Detail:   "The legacy provider namespace \"-\" can be used only with hostname " + DefaultRegistryHost.ForDisplay() + ".",
		})
		return Provider{}, diags
	}

	// Due to how plugin executables are named and provider git repositories
	// are conventionally named, it's a reasonable and
	// apparently-somewhat-common user error to incorrectly use the
	// "terraform-provider-" prefix in a provider source address. There is
	// no good reason for a provider to have the prefix "terraform-" anyway,
	// so we've made that invalid from the start both so we can give feedback
	// to provider developers about the terraform- prefix being redundant
	// and give specialized feedback to folks who incorrectly use the full
	// terraform-provider- prefix to help them self-correct.
	const redundantPrefix = "terraform-"
	const userErrorPrefix = "terraform-provider-"
	if strings.HasPrefix(ret.Type, redundantPrefix) {
		if strings.HasPrefix(ret.Type, userErrorPrefix) {
			// Likely user error. We only return this specialized error if
			// whatever is after the prefix would otherwise be a
			// syntactically-valid provider type, so we don't end up advising
			// the user to try something that would be invalid for another
			// reason anyway.
			// (This is mainly just for robustness, because the validation
			// we already did above should've rejected most/all ways for
			// the suggestedType to end up invalid here.)
			suggestedType := ret.Type[len(userErrorPrefix):]
			if _, err := ParseProviderPart(suggestedType); err == nil {
				suggestedAddr := ret
				suggestedAddr.Type = suggestedType
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid provider type",
					fmt.Sprintf("Provider source %q has a type with the prefix %q, which isn't valid. Although that prefix is often used in the names of version control repositories for Terraform providers, provider source strings should not include it.\n\nDid you mean %q?", ret.ForDisplay(), userErrorPrefix, suggestedAddr.ForDisplay()),
				))
				return Provider{}, diags
			}
		}
		// Otherwise, probably instead an incorrectly-named provider, perhaps
		// arising from a similar instinct to what causes there to be
		// thousands of Python packages on PyPI with "python-"-prefixed
		// names.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid provider type",
			fmt.Sprintf("Provider source %q has a type with the prefix %q, which isn't allowed because it would be redundant to name a Terraform provider with that prefix. If you are the author of this provider, rename it to not include the prefix.", ret, redundantPrefix),
		))
		return Provider{}, diags
	}

	return ret, diags
}

// MustParseProviderSourceString is a wrapper around ParseProviderSourceString that panics if
// it returns an error.
func MustParseProviderSourceString(str string) Provider {
	result, diags := ParseProviderSourceString(str)
	if diags.HasErrors() {
		panic(diags.Err().Error())
	}
	return result
}

// ParseProviderPart processes an addrs.Provider namespace or type string
// provided by an end-user, producing a normalized version if possible or
// an error if the string contains invalid characters.
//
// A provider part is processed in the same way as an individual label in a DNS
// domain name: it is transformed to lowercase per the usual DNS case mapping
// and normalization rules and may contain only letters, digits, and dashes.
// Additionally, dashes may not appear at the start or end of the string.
//
// These restrictions are intended to allow these names to appear in fussy
// contexts such as directory/file names on case-insensitive filesystems,
// repository names on GitHub, etc. We're using the DNS rules in particular,
// rather than some similar rules defined locally, because the hostname part
// of an addrs.Provider is already a hostname and it's ideal to use exactly
// the same case folding and normalization rules for all of the parts.
//
// In practice a provider type string conventionally does not contain dashes
// either. Such names are permitted, but providers with such type names will be
// hard to use because their resource type names will not be able to contain
// the provider type name and thus each resource will need an explicit provider
// address specified. (A real-world example of such a provider is the
// "google-beta" variant of the GCP provider, which has resource types that
// start with the "google_" prefix instead.)
//
// It's valid to pass the result of this function as the argument to a
// subsequent call, in which case the result will be identical.
func ParseProviderPart(given string) (string, error) {
	if len(given) == 0 {
		return "", fmt.Errorf("must have at least one character")
	}

	// We're going to process the given name using the same "IDNA" library we
	// use for the hostname portion, since it already implements the case
	// folding rules we want.
	//
	// The idna library doesn't expose individual label parsing directly, but
	// once we've verified it doesn't contain any dots we can just treat it
	// like a top-level domain for this library's purposes.
	if strings.ContainsRune(given, '.') {
		return "", fmt.Errorf("dots are not allowed")
	}

	// We don't allow names containing multiple consecutive dashes, just as
	// a matter of preference: they look weird, confusing, or incorrect.
	// This also, as a side-effect, prevents the use of the "punycode"
	// indicator prefix "xn--" that would cause the IDNA library to interpret
	// the given name as punycode, because that would be weird and unexpected.
	if strings.Contains(given, "--") {
		return "", fmt.Errorf("cannot use multiple consecutive dashes")
	}

	result, err := idna.Lookup.ToUnicode(given)
	if err != nil {
		return "", fmt.Errorf("must contain only letters, digits, and dashes, and may not use leading or trailing dashes")
	}

	return result, nil
}

// MustParseProviderPart is a wrapper around ParseProviderPart that panics if
// it returns an error.
func MustParseProviderPart(given string) string {
	result, err := ParseProviderPart(given)
	if err != nil {
		panic(err.Error())
	}
	return result
}

// IsProviderPartNormalized compares a given string to the result of ParseProviderPart(string)
func IsProviderPartNormalized(str string) (bool, error) {
	normalized, err := ParseProviderPart(str)
	if err != nil {
		return false, err
	}
	if str == normalized {
		return true, nil
	}
	return false, nil
}
