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
	// No children is easy: we're already merged!
	if len(t.Children) == 0 {
		return t.Config, nil
	}

	// Depth-first, merge all the children first.
	childConfigs := make([]*Config, len(t.Children))
	for i, ct := range t.Children {
		c, err := ct.Flatten()
		if err != nil {
			return nil, err
		}

		childConfigs[i] = c
	}

	// Merge all the children in order
	config := childConfigs[0]
	childConfigs = childConfigs[1:]
	for _, config2 := range childConfigs {
		var err error
		config, err = Merge(config, config2)
		if err != nil {
			return nil, err
		}
	}

	// Merge the final merged child config with our own
	return Merge(config, t.Config)
}
