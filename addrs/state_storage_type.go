package addrs

// StateStorageType represents a type of state storage. A state storage type
// is instantiated by combining it with a suitable configuration.
type StateStorageType struct {
	ProviderType ProviderType
	Name         string
}
