package config

import (
	"fmt"
)

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
		config, err = mergeConfig(config, config2)
		if err != nil {
			return nil, err
		}
	}

	// Merge the final merged child config with our own
	return mergeConfig(config, t.Config)
}

func mergeConfig(c1, c2 *Config) (*Config, error) {
	c := new(Config)

	// Merge variables: Variable merging is quite simple. Set fields in
	// later set variables override those earlier.
	c.Variables = c1.Variables
	for k, v2 := range c2.Variables {
		v1, ok := c.Variables[k]
		if ok {
			if v2.Default == "" {
				v2.Default = v1.Default
			}
			if v2.Description == "" {
				v2.Description = v1.Description
			}
		}

		c.Variables[k] = v2
	}

	// Merge provider configs: If they collide, we just take the latest one
	// for now. In the future, we might provide smarter merge functionality.
	c.ProviderConfigs = make(map[string]*ProviderConfig)
	for k, v := range c1.ProviderConfigs {
		c.ProviderConfigs[k] = v
	}
	for k, v := range c2.ProviderConfigs {
		c.ProviderConfigs[k] = v
	}

	// Merge resources: If they collide, we just take the latest one
	// for now. In the future, we might provide smarter merge functionality.
	resources := make(map[string]*Resource)
	for _, r := range c1.Resources {
		id := fmt.Sprintf("%s[%s]", r.Type, r.Name)
		resources[id] = r
	}
	for _, r := range c2.Resources {
		id := fmt.Sprintf("%s[%s]", r.Type, r.Name)
		resources[id] = r
	}

	c.Resources = make([]*Resource, 0, len(resources))
	for _, r := range resources {
		c.Resources = append(c.Resources, r)
	}

	return c, nil
}
