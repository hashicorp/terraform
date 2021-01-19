package stressgen

import (
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressaddr"
	"github.com/zclconf/go-cty/cty"
)

// GenerateModifiedConfig produces a new configuration which is a valid
// modification of the reciever, using the given modification address as
// a random seed for deciding what to change.
func (c *Config) GenerateModifiedConfig(modAddr stressaddr.ModConfig) *Config {
	rnd := newRand(modAddr.RandomSeed())
	addr := c.Addr.NewMod(modAddr)
	ns := NewNamespace()
	reg := NewRootRegistry()

	objs := make([]ConfigObject, 0, len(c.Objects))
	insts := make([]ConfigObjectInstance, 0, len(c.ObjectInstances))
	for _, old := range c.Objects {
		new := old.GenerateModified(rnd, ns)
		if new == nil {
			// This represents removing the object altogether.
			continue
		}
		objs = append(objs, new)

		// This is tricky: if the generated object is representing an input
		// variable that the caller is expected to set, we need to choose
		// a value for it _before_ we instantiate the object, so that it
		// can take into account the value we've chosen when it decides its
		// own expected value.
		if cv, ok := new.(*ConfigVariable); ok && cv.CallerWillSet {
			// TODO: Should we have the possibility of preserving a previous
			// value here? Currently we just regenerate a new value every
			// time because we can't see what we previously chose here,
			// but this means that anything derived from a root input
			// variable will churn on every step in a series.
			chosenVal := cty.StringVal(ns.GenerateLongName(rnd))
			reg.RegisterVariableValue(cv.Addr, chosenVal)
		}

		newInst := new.Instantiate(reg)

		insts = append(insts, newInst)

		// TODO: With a relatively low likelihood, potentially generate
		// new blocks too.
	}

	return &Config{
		Addr:            addr,
		Objects:         objs,
		ObjectInstances: insts,
		Namespace:       ns,
		Registry:        reg,
	}
}
