package config

// configTree represents a tree of configurations where the root is the
// first file and its children are the configurations it has imported.
type configTree struct {
	Path     string
	Config   *Config
	Children []*configTree
}

// Flatten flattens the entire tree down to a single merged Config
// structure.
func (t *configTree) Flatten() (*Config, error) {
	return t.Config, nil
}
