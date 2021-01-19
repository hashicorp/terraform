package stressgen

import (
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressaddr"
)

// GenerateConfigSeries uses a random number generator seeded with a value
// derived from the given stressaddr.ConfigSeries to construct a series of
// random but valid Terraform configurations, each subsequent one of which
// is a valid edit of its predecessor.
//
// This is the main entry point for this package, for generating test cases to
// be run by the stresstest graph test harness.
//
// All randomness in the generator is derived by using the given address as
// a seed, so passing the same address will produce the same result as long
// as the generator code itself hasn't been modified in the meantime. (This
// also includes modifications to the Go math/rand package, whose generator
// we use here.)
func GenerateConfigSeries(addr stressaddr.ConfigSeries) *ConfigSeries {
	rnd := newRand(addr.RandomSeed())

	// Randomly-generating a config address implicitly decides all of the
	// following, due to these being packed into the address:
	// - The initial configuration address
	// - How many subsequent modification configs will be generated
	// - The addresses of each of those modifications.
	configAddr := stressaddr.RandomConfig(rnd)
	steps := generateConfigSteps(configAddr)
	return &ConfigSeries{
		Addr:  addr,
		Steps: steps,
	}

}

// GenerateConfig constructs a specific configuration using the given address.
//
// The main purpose of this method is to reconstruct a specific configuration
// you have an address for even if you don't know the address of the series
// it was created as a part of. If you know the series address then you can
// use GenerateConfigSeries instead.
func GenerateConfig(addr stressaddr.Config) *Config {
	// Because our main path is to generate a whole series, and because we
	// need to walk through all the same steps it would do to arrive at the
	// same result, this is really just a different way to do what
	// GenerateConfigSeries would do, except that it discards all but the
	// final generated configuration.
	steps := generateConfigSteps(addr)
	return steps[len(steps)-1]
}

// generateConfigSeries is the main body of both GenerateConfigSeries and
// GenerateConfig, both of which need to walk through generating an
// initial configuration and then zero or more modified ones. They just then
// each use the result in different ways.
func generateConfigSteps(addr stressaddr.Config) []*Config {
	// We must always have at least one initial configuration to plan and apply,
	// establishing objects which we might then modify in subsequent steps.
	initial := GenerateInitialConfig(addr.Start)
	ret := make([]*Config, 1, len(addr.Mods)+1)
	ret[0] = initial

	prev := initial
	for _, modAddr := range addr.Mods {
		next := prev.GenerateModifiedConfig(modAddr)
		ret = append(ret, next)
		prev = next
	}

	return ret
}
