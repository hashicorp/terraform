package addrs

import svchost "github.com/hashicorp/terraform-svchost"

// ProviderType encapsulates a single provider type. In the future this will be
// extended to include additional fields including Namespace and SourceHost
type ProviderType struct {
	Type      string
	Namespace string
	Hostname  svchost.Hostname
}

func (pt ProviderType) String() string {
	return pt.Hostname.ForDisplay() + "/" + pt.Namespace + "/" + pt.Type
}

// NewDefaultProviderType returns the default address of a HashiCorp-maintained,
// Registry-hosted provider.
func NewDefaultProviderType(name string) ProviderType {
	return ProviderType{
		Type:      name,
		Namespace: "hashicorp",
		Hostname:  "registry.terraform.io",
	}
}

// NewLegacyProviderType returns a mock address for a provider.
// This will be removed when ProviderType is fully integrated.
func NewLegacyProviderType(name string) ProviderType {
	return ProviderType{
		Type:      name,
		Namespace: "-",
		Hostname:  "registry.terraform.io",
	}
}

// LegacyString returns the provider type, which is frequently used
// interchangeably with provider name. This function can and should be removed
// when provider type is fully integrated.
func (pt ProviderType) LegacyString() string {
	return pt.Type
}
