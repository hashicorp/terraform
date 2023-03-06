package addrs

import (
	"github.com/hashicorp/hcl/v2"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Provider encapsulates a single provider type. In the future this will be
// extended to include additional fields including Namespace and SourceHost
type Provider = tfaddr.Provider

// DefaultProviderRegistryHost is the hostname used for provider addresses that do
// not have an explicit hostname.
const DefaultProviderRegistryHost = tfaddr.DefaultProviderRegistryHost

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

// legacyImpliedProviders are providers that older versions of Terraform
// would allow to be used without an explicit required_providers entry,
// for backward compatibility with modules written for older Terraform
// versions that didn't yet support third-party providers and so were
// able to just assume _everything_ was an official provider.
//
// Although we've supported explicit provider source addresses since
// Terraform v0.13, versions up to Terraform v1.4 just treated any unqualified
// provider as an assumed-official provider and so in practice this map
// includes official providers that were published after Terraform v0.13 but
// before v1.4. Any new official providers published after Terraform CLI v1.4.0
// should not be added here because there is no possibility of older modules
// referring to them and therefore no backward-compatibility concern for them.
var legacyImpliedProviders = map[string]Provider{
	"ad":              NewOfficialProvider("ad"),
	"archive":         NewOfficialProvider("archive"),
	"aws":             NewOfficialProvider("aws"),
	"awscc":           NewOfficialProvider("awscc"),
	"azuread":         NewOfficialProvider("azuread"),
	"azurerm":         NewOfficialProvider("azurerm"),
	"azurestack":      NewOfficialProvider("azurestack"),
	"boundary":        NewOfficialProvider("boundary"),
	"cloudinit":       NewOfficialProvider("cloudinit"),
	"consul":          NewOfficialProvider("consul"),
	"dns":             NewOfficialProvider("dns"),
	"external":        NewOfficialProvider("external"),
	"google":          NewOfficialProvider("google"),
	"google-beta":     NewOfficialProvider("google-beta"),
	"googleworkspace": NewOfficialProvider("googleworkspace"),
	"hashicups":       NewOfficialProvider("hashicups"),
	"hcp":             NewOfficialProvider("hcp"),
	"hcs":             NewOfficialProvider("hcs"),
	"helm":            NewOfficialProvider("helm"),
	"http":            NewOfficialProvider("http"),
	"kubernetes":      NewOfficialProvider("kubernetes"),
	"local":           NewOfficialProvider("local"),
	"nomad":           NewOfficialProvider("nomad"),
	"null":            NewOfficialProvider("null"),
	"opc":             NewOfficialProvider("opc"),
	"oraclepaas":      NewOfficialProvider("oraclepaas"),
	"random":          NewOfficialProvider("random"),
	"salesforce":      NewOfficialProvider("salesforce"),
	"template":        NewOfficialProvider("template"),
	"terraform":       NewBuiltInProvider("terraform"),
	"tfcoremock":      NewOfficialProvider("tfcoremock"),
	"tfe":             NewOfficialProvider("tfe"),
	"time":            NewOfficialProvider("time"),
	"tls":             NewOfficialProvider("tls"),
	"vault":           NewOfficialProvider("vault"),
	"vsphere":         NewOfficialProvider("vsphere"),
}

// IsOfficialProvider returns true if and only if the given provider address
// belongs to the official public registry's "hashicorp" namespace.
//
// Packages for providers in that namespace should also typically be signed
// with a HashiCorp private key, but this function only deals with the
// address of the provider and not its packages and so it cannot guarantee
// that any particular package belonging to an official provider is correctly
// signed. The provider installer is responsible for verifying signatures
// during installation.
func IsOfficialProvider(addr Provider) bool {
	return addr.Hostname == DefaultProviderRegistryHost && addr.Namespace == "hashicorp"
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
	return tfaddr.NewProvider(hostname, namespace, typeName)
}

// ImpliedProviderForUnqualifiedType implements the backward-compatibility rules
// for handling modules that lack explicit required_providers declarations
// because they were written before those were supported or before they were
// required.
//
// Provider type names that were supported for implicit selection in Terraform
// v1.3 or earlier will return the corresponding explicit provider source
// address and true. Name that was not available as an implied provider before
// Terraform v1.4 is invalid and so will return a zero-value provider and
// false to indicate that the result isn't valid.
//
// All provider addresses successfully returned from this function will be
// so-called "official providers" belonging to the "hashicorp" namespace on
// the public registry, except for the special provider type "terraform" which
// is treated as "terraform.io/builtin/terraform".
func ImpliedProviderForUnqualifiedType(typeName string) (addr Provider, ok bool) {
	addr, ok = legacyImpliedProviders[typeName]
	return addr, ok
}

// NewOfficialProvider returns the default address of a HashiCorp-maintained,
// Registry-hosted provider.
func NewOfficialProvider(name string) Provider {
	return tfaddr.Provider{
		Type:      MustParseProviderPart(name),
		Namespace: "hashicorp",
		Hostname:  DefaultProviderRegistryHost,
	}
}

// NewBuiltInProvider returns the address of a "built-in" provider. See
// the docs for Provider.IsBuiltIn for more information.
func NewBuiltInProvider(name string) Provider {
	return tfaddr.Provider{
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
		Hostname:  DefaultProviderRegistryHost,
	}
}

// ParseProviderSourceString parses a value of the form expected in the "source"
// argument of a required_providers entry and returns the corresponding
// fully-qualified provider address. This is intended primarily to parse the
// FQN-like strings returned by terraform-config-inspect.
//
// The following are valid source string formats:
//
//   - name
//   - namespace/name
//   - hostname/namespace/name
func ParseProviderSourceString(str string) (tfaddr.Provider, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	ret, err := tfaddr.ParseProviderSource(str)
	if pe, ok := err.(*tfaddr.ParserError); ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  pe.Summary,
			Detail:   pe.Detail,
		})
		return ret, diags
	}

	if !ret.HasKnownNamespace() {
		ret.Namespace = "hashicorp"
	}

	return ret, nil
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
