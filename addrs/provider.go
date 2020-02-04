package addrs

import (
	"fmt"
	"regexp"
	"strings"

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

const DefaultRegistryHost = "registry.terraform.io"

var (
	ValidProviderName = regexp.MustCompile("^[a-zA-Z0-9_-]+$")
)

// String returns an FQN string, indended for use in output.
func (pt Provider) String() string {
	return pt.Hostname.ForDisplay() + "/" + pt.Namespace + "/" + pt.Type
}

// NewDefaultProvider returns the default address of a HashiCorp-maintained,
// Registry-hosted provider.
func NewDefaultProvider(name string) Provider {
	return Provider{
		Type:      name,
		Namespace: "hashicorp",
		Hostname:  DefaultRegistryHost,
	}
}

// NewLegacyProvider returns a mock address for a provider.
// This will be removed when ProviderType is fully integrated.
func NewLegacyProvider(name string) Provider {
	// This is intended to catch provider names with aliases, such as "null.foo"
	if !ValidProviderName.MatchString(name) {
		panic("invalid provider name")
	}

	return Provider{
		Type:      name,
		Namespace: "-",
		Hostname:  DefaultRegistryHost,
	}
}

// LegacyString returns the provider type, which is frequently used
// interchangeably with provider name. This function can and should be removed
// when provider type is fully integrated. As a safeguard for future
// refactoring, this function panics if the Provider is not a legacy provider.
func (pt Provider) LegacyString() string {
	if pt.Namespace != "-" {
		panic("not a legacy Provider")
	}
	return pt.Type
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
	name := parts[len(parts)-1]
	if !ValidProviderName.MatchString(name) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider type",
			Detail:   fmt.Sprintf(`Invalid provider type %q in source %q: must be a provider type name"`, name, str),
		})
		return ret, diags
	}
	ret.Type = name
	ret.Hostname = DefaultRegistryHost

	if len(parts) == 1 {
		// FIXME: update this to NewDefaultProvider in the provider source release
		return NewLegacyProvider(parts[0]), diags
	}

	if len(parts) >= 2 {
		// the namespace is always the second-to-last part
		namespace := parts[len(parts)-2]
		if !ValidProviderName.MatchString(namespace) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider namespace",
				Detail:   fmt.Sprintf(`Invalid provider namespace %q in source %q: must be a valid Registry Namespace"`, namespace, str),
			})
			return Provider{}, diags
		}
		ret.Namespace = namespace
	}

	// Final Case: 3 parts
	if len(parts) == 3 {
		// the namespace is always the first part in a three-part source string
		hn, err := svchost.ForComparison(parts[0])
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider source hostname",
				Detail:   fmt.Sprintf(`Invalid provider source hostname namespace %q in source %q: must be a valid Registry Namespace"`, hn, str),
			})
			return Provider{}, diags
		}
		ret.Hostname = hn
	}

	return ret, diags
}
