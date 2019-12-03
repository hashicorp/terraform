package addrs

// ProviderType encapsulates a single provider type. In the future this will be
// extended to include additional fields including Namespace and SourceHost
type ProviderType struct {
	Type      string
	Namespace string
	Hostname  string
}

func (pt ProviderType) String() string {
	return pt.Hostname + "/" + pt.Namespace + "/" + pt.Type
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
