package module

// Module represents the metadata for a single module.
type Module struct {
	Name      string
	Source    string
	Version   string
	Providers map[string]string
}
