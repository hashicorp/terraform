package config

// Merge merges two configurations into a single configuration.
//
// Merge allows for the two configurations to have duplicate resources,
// because the resources will be merged. This differs from a single
// Config which must only have unique resources.
func Merge(c1, c2 *Config) (*Config, error) {
	c := new(Config)

	// Merge unknown keys
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

	// Merge Atlas configuration. This is a dumb one overrides the other
	// sort of merge.
	c.Atlas = c1.Atlas
	if c2.Atlas != nil {
		c.Atlas = c2.Atlas
	}

	// Merge the Terraform configuration
	if c1.Terraform != nil {
		c.Terraform = c1.Terraform
		if c2.Terraform != nil {
			c.Terraform.Merge(c2.Terraform)
		}
	} else {
		c.Terraform = c2.Terraform
	}

	// NOTE: Everything below is pretty gross. Due to the lack of generics
	// in Go, there is some hoop-jumping involved to make this merging a
	// little more test-friendly and less repetitive. Ironically, making it
	// less repetitive involves being a little repetitive, but I prefer to
	// be repetitive with things that are less error prone than things that
	// are more error prone (more logic). Type conversions to an interface
	// are pretty low-error.

	var m1, m2, mresult []merger

	// Modules
	m1 = make([]merger, 0, len(c1.Modules))
	m2 = make([]merger, 0, len(c2.Modules))
	for _, v := range c1.Modules {
		m1 = append(m1, v)
	}
	for _, v := range c2.Modules {
		m2 = append(m2, v)
	}
	mresult = mergeSlice(m1, m2)
	if len(mresult) > 0 {
		c.Modules = make([]*Module, len(mresult))
		for i, v := range mresult {
			c.Modules[i] = v.(*Module)
		}
	}

	// Outputs
	m1 = make([]merger, 0, len(c1.Outputs))
	m2 = make([]merger, 0, len(c2.Outputs))
	for _, v := range c1.Outputs {
		m1 = append(m1, v)
	}
	for _, v := range c2.Outputs {
		m2 = append(m2, v)
	}
	mresult = mergeSlice(m1, m2)
	if len(mresult) > 0 {
		c.Outputs = make([]*Output, len(mresult))
		for i, v := range mresult {
			c.Outputs[i] = v.(*Output)
		}
	}

	// Provider Configs
	m1 = make([]merger, 0, len(c1.ProviderConfigs))
	m2 = make([]merger, 0, len(c2.ProviderConfigs))
	for _, v := range c1.ProviderConfigs {
		m1 = append(m1, v)
	}
	for _, v := range c2.ProviderConfigs {
		m2 = append(m2, v)
	}
	mresult = mergeSlice(m1, m2)
	if len(mresult) > 0 {
		c.ProviderConfigs = make([]*ProviderConfig, len(mresult))
		for i, v := range mresult {
			c.ProviderConfigs[i] = v.(*ProviderConfig)
		}
	}

	// Resources
	m1 = make([]merger, 0, len(c1.Resources))
	m2 = make([]merger, 0, len(c2.Resources))
	for _, v := range c1.Resources {
		m1 = append(m1, v)
	}
	for _, v := range c2.Resources {
		m2 = append(m2, v)
	}
	mresult = mergeSlice(m1, m2)
	if len(mresult) > 0 {
		c.Resources = make([]*Resource, len(mresult))
		for i, v := range mresult {
			c.Resources[i] = v.(*Resource)
		}
	}

	// Variables
	m1 = make([]merger, 0, len(c1.Variables))
	m2 = make([]merger, 0, len(c2.Variables))
	for _, v := range c1.Variables {
		m1 = append(m1, v)
	}
	for _, v := range c2.Variables {
		m2 = append(m2, v)
	}
	mresult = mergeSlice(m1, m2)
	if len(mresult) > 0 {
		c.Variables = make([]*Variable, len(mresult))
		for i, v := range mresult {
			c.Variables[i] = v.(*Variable)
		}
	}

	// Local Values
	// These are simpler than the other config elements because they are just
	// flat values and so no deep merging is required.
	if localsCount := len(c1.Locals) + len(c2.Locals); localsCount != 0 {
		// Explicit length check above because we want c.Locals to remain
		// nil if the result would be empty.
		c.Locals = make([]*Local, 0, len(c1.Locals)+len(c2.Locals))
		c.Locals = append(c.Locals, c1.Locals...)
		c.Locals = append(c.Locals, c2.Locals...)
	}

	return c, nil
}

// merger is an interface that must be implemented by types that are
// merge-able. This simplifies the implementation of Merge for the various
// components of a Config.
type merger interface {
	mergerName() string
	mergerMerge(merger) merger
}

// mergeSlice merges a slice of mergers.
func mergeSlice(m1, m2 []merger) []merger {
	r := make([]merger, len(m1), len(m1)+len(m2))
	copy(r, m1)

	m := map[string]struct{}{}
	for _, v2 := range m2 {
		// If we already saw it, just append it because its a
		// duplicate and invalid...
		name := v2.mergerName()
		if _, ok := m[name]; ok {
			r = append(r, v2)
			continue
		}
		m[name] = struct{}{}

		// Find an original to override
		var original merger
		originalIndex := -1
		for i, v := range m1 {
			if v.mergerName() == name {
				originalIndex = i
				original = v
				break
			}
		}

		var v merger
		if original == nil {
			v = v2
		} else {
			v = original.mergerMerge(v2)
		}

		if originalIndex == -1 {
			r = append(r, v)
		} else {
			r[originalIndex] = v
		}
	}

	return r
}
