package addrs

import (
	svchost "github.com/hashicorp/terraform-svchost"
)

// Provider encapsulates a single provider type. In the future this will be
// extended to include additional fields including Namespace and SourceHost
type Provider struct {
	Type      string
	Namespace string
	Hostname  svchost.Hostname
}

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
		Hostname:  "registry.terraform.io",
	}
}

// NewLegacyProvider returns a mock address for a provider.
// This will be removed when ProviderType is fully integrated.
func NewLegacyProvider(name string) Provider {
	return Provider{
		Type:      name,
		Namespace: "-",
		Hostname:  "registry.terraform.io",
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
