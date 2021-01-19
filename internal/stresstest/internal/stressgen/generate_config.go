package stressgen

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressaddr"
	"github.com/zclconf/go-cty/cty"
)

// GenerateInitialConfig uses a random number generator seeded with a value
// derived from the given stressaddr.StartConfig to construct a random but
// valid initial Terraform configuration.
//
// All randomness in the generator is derived by using the given address as
// a seed, so passing the same address will produce the same result as long
// as the generator code itself hasn't been modified in the meantime. (This
// also includes modifications to the Go math/rand package, whose generator
// we use here.)
//
// Note that Config isn't actually the top-level object for a test case. If
// you are trying to randomly generate a full test case then you should call
// GenerateConfigSeries, which calls GenerateConfig to establish an initial
// configuration and then randomly generates modifications to it to simulate
// a configuration being maintained over time.
func GenerateInitialConfig(addr stressaddr.StartConfig) *Config {
	rnd := newRand(addr.RandomSeed())
	ns := NewNamespace()
	reg := NewRootRegistry()

	// We'll create no more than 24 objects because more objects will tend
	// to make the runtime longer. It is possible that a larger graph would
	// make it more likely to generate obnoxious graph shapes, but 24 objects
	// seems like it ought to be enough to make room for various interesting
	// permutations.
	objCount := rnd.Intn(25)
	objs := make([]ConfigObject, 0, objCount+1)
	insts := make([]ConfigObjectInstance, 0, objCount+1)

	// We always need a boilerplate object.
	boilerplate := &ConfigBoilerplate{
		ModuleAddr: addrs.RootModule,
		Providers: map[string]addrs.Provider{
			"stressful": addrs.MustParseProviderSourceString("terraform.io/stresstest/stressful"),
		},
	}
	objs = append(objs, boilerplate)
	insts = append(insts, boilerplate)

	// Each object we generate can potentially make use of registry entries
	// created by any object that came before it, and so we can generate
	// a variety of interesting graph shapes. Note also that this prevents
	// creating any reference cycles, because an object can't refer to any
	// object that would be added _after_ it. (Not all graph edges are created
	// by references though, so object generators must still be careful to
	// avoid generating cycles in other ways.)
	for i := 0; i < objCount; i++ {
		obj := GenerateConfigObject(rnd, ns)
		objs = append(objs, obj)

		// This is tricky: if the generated object is representing an input
		// variable that the caller is expected to set, we need to choose
		// a value for it _before_ we instantiate the object, so that it
		// can take into account the value we've chosen when it decides its
		// own expected value.
		if cv, ok := obj.(*ConfigVariable); ok && cv.CallerWillSet {
			chosenVal := cty.StringVal(ns.GenerateLongName(rnd))
			reg.RegisterVariableValue(cv.Addr, chosenVal)
		}

		inst := obj.Instantiate(reg)
		insts = append(insts, inst)
	}

	return &Config{
		Addr:            stressaddr.FixedConfig(addr),
		Objects:         objs,
		ObjectInstances: insts,
		Namespace:       ns,
		Registry:        reg,
	}
}
