package addrs

// ProviderType represents a provider type. Each ProviderConfig represents an
// instance of a provider type by associating a configuration with it.
type ProviderType struct {
	Hostname  string
	Namespace string
	Type      string
}
