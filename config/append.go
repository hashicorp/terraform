package config

// Append appends one configuration to another.
//
// Append assumes that both configurations will not have
// conflicting variables, resources, etc. If they do, the
// problems will be caught in the validation phase.
//
// It is possible that c1, c2 on their own are not valid. For
// example, a resource in c2 may reference a variable in c1. But
// together, they would be valid.
func Append(c1, c2 *Config) (*Config, error) {
	c := new(Config)

	// Append unknown keys, but keep them unique since it is a set
	unknowns := make(map[string]struct{})
	for _, k := range c1.unknownKeys {
		_, present := unknowns[k]
		if !present {
			unknowns[k] = struct{}{}
			c.unknownKeys = append(c.unknownKeys, k)
		}
	}

	for _, k := range c2.unknownKeys {
		_, present := unknowns[k]
		if !present {
			unknowns[k] = struct{}{}
			c.unknownKeys = append(c.unknownKeys, k)
		}
	}

	c.Atlas = c1.Atlas
	if c2.Atlas != nil {
		c.Atlas = c2.Atlas
	}

	if len(c1.Modules) > 0 || len(c2.Modules) > 0 {
		c.Modules = make(
			[]*Module, 0, len(c1.Modules)+len(c2.Modules))
		c.Modules = append(c.Modules, c1.Modules...)
		c.Modules = append(c.Modules, c2.Modules...)
	}

	if len(c1.Outputs) > 0 || len(c2.Outputs) > 0 {
		c.Outputs = make(
			[]*Output, 0, len(c1.Outputs)+len(c2.Outputs))
		c.Outputs = append(c.Outputs, c1.Outputs...)
		c.Outputs = append(c.Outputs, c2.Outputs...)
	}

	if len(c1.ProviderConfigs) > 0 || len(c2.ProviderConfigs) > 0 {
		c.ProviderConfigs = make(
			[]*ProviderConfig,
			0, len(c1.ProviderConfigs)+len(c2.ProviderConfigs))
		c.ProviderConfigs = append(c.ProviderConfigs, c1.ProviderConfigs...)
		c.ProviderConfigs = append(c.ProviderConfigs, c2.ProviderConfigs...)
	}

	if len(c1.Resources) > 0 || len(c2.Resources) > 0 {
		c.Resources = make(
			[]*Resource,
			0, len(c1.Resources)+len(c2.Resources))
		c.Resources = append(c.Resources, c1.Resources...)
		c.Resources = append(c.Resources, c2.Resources...)
	}

	if len(c1.Variables) > 0 || len(c2.Variables) > 0 {
		c.Variables = make(
			[]*Variable, 0, len(c1.Variables)+len(c2.Variables))
		c.Variables = append(c.Variables, c1.Variables...)
		c.Variables = append(c.Variables, c2.Variables...)
	}

	return c, nil
}
