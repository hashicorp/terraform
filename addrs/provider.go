package addrs

import (
	"errors"

	"github.com/hashicorp/terraform-registry-address"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/tfdiags"
)

// Provider encapsulates a single provider type. In the future this will be
// extended to include additional fields including Namespace and SourceHost
type Provider = tfaddr.Provider

// DefaultRegistryHost is the hostname used for provider addresses that do
// not have an explicit hostname.
const DefaultRegistryHost = tfaddr.DefaultRegistryHost

// BuiltInProviderHost is the pseudo-hostname used for the "built-in" provider
// namespace. Built-in provider addresses must also have their namespace set
// to BuiltInProviderNamespace in order to be considered as built-in.
const BuiltInProviderHost = tfaddr.BuiltInProviderHost

// BuiltInProviderNamespace is the provider namespace used for "built-in"
// providers. Built-in provider addresses must also have their hostname
// set to BuiltInProviderHost in order to be considered as built-in.
//
// The this namespace is literally named "builtin", in the hope that users
// who see FQNs containing this will be able to infer the way in which they are
// special, even if they haven't encountered the concept formally yet.
const BuiltInProviderNamespace = tfaddr.BuiltInProviderNamespace

// LegacyProviderNamespace is the special string used in the Namespace field
// of type Provider to mark a legacy provider address. This special namespace
// value would normally be invalid, and can be used only when the hostname is
// DefaultRegistryHost because that host owns the mapping from legacy name to
// FQN.
const LegacyProviderNamespace = tfaddr.LegacyProviderNamespace

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
	return Provider(tfaddr.NewProvider(hostname, namespace, typeName))
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
	return Provider(tfaddr.ImpliedProviderForUnqualifiedType(typeName))
}

// NewDefaultProvider returns the default address of a HashiCorp-maintained,
// Registry-hosted provider.
func NewDefaultProvider(name string) Provider {
	return Provider(tfaddr.NewDefaultProvider(name))
}

// NewBuiltInProvider returns the address of a "built-in" provider. See
// the docs for Provider.IsBuiltIn for more information.
func NewBuiltInProvider(name string) Provider {
	return Provider(tfaddr.NewBuiltInProvider(name))
}

// NewLegacyProvider returns a mock address for a provider.
// This will be removed when ProviderType is fully integrated.
func NewLegacyProvider(name string) Provider {
	return Provider(tfaddr.NewLegacyProvider(name))
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
	diags := make(tfdiags.Diagnostics, 0)

	pAddr, err := tfaddr.ParseAndInferProviderSourceString(str)
	if err != nil {
		parserErr := &tfaddr.ParserError{}
		if errors.As(err, &parserErr) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				parserErr.Summary,
				parserErr.Detail,
			))
		} else {
			diags = diags.Append(tfdiags.FormatError(err))
		}
	}

	return Provider(pAddr), diags
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
	return tfaddr.ParseProviderPart(given)
}

// MustParseProviderPart is a wrapper around ParseProviderPart that panics if
// it returns an error.
func MustParseProviderPart(given string) string {
	return tfaddr.MustParseProviderPart(given)
}

// IsProviderPartNormalized compares a given string to the result of ParseProviderPart(string)
func IsProviderPartNormalized(str string) (bool, error) {
	return tfaddr.IsProviderPartNormalized(str)
}
