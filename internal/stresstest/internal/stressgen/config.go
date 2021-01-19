package stressgen

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressaddr"
	"github.com/hashicorp/terraform/states"
)

// Config represents a generated configuration.
//
// It only directly refers to the generated root module, but that module might
// in turn contain references to child modules via module call objects.
//
// This type and most of its descendents have exported fields just because
// this package is aimed at testing use-cases and having them exported tends
// to make debugging easier. With that said, external callers should generally
// not modify any data in those exported fields, and should instead prefer to
// use the methods on these types that know how to derive new objects while
// keeping all of the expected invariants maintained.
//
// The top-level object representing a test case is ConfigSeries, which is a
// sequence of Config instances that will be planned, applied, and verified in
// order. Config therefore represents only a single step in a test case.
type Config struct {
	// Addr is an identifier for this particular generated configuration, which
	// a caller can use to rebuild the same configuration as long as nothing
	// in the config generator code has changed in the meantime.
	Addr stressaddr.Config

	// A generated configuration is made from a series of "objects", each of
	// which typically corresponds to one configuration block when we serialize
	// the configuration into normal Terraform language input.
	Objects []ConfigObject

	// In the root module, each configuration object always has exactly one
	// instance, and so ObjectInstances is always the same length as Objects
	// and elements of ObjectInstances correspond with elements of Objects.
	//
	// The instance of each object is what contains the prediction for the
	// expected values within an object. The separation of ConfigObject and
	// ConfigObjectInstance seems unnecessary at the root because they are
	// always one-to-one, but this distinction exists to allow for
	// multi-instanced child modules where therefore any objects in those
	// modules would also be multi-instanced.
	ObjectInstances []ConfigObjectInstance

	// Namespace summarizes the names that are used within the module. Although
	// this object can be modified in principle, by the time a Namespace is
	// assigned into a Config it should be treated as immutable by convention.
	//
	// Most of the items recorded in a Namespace are internal to the module,
	// but the recorded input variables will help the test harness generate
	// valid values to successfully call the module.
	//
	// Some methods of Config combine data from Namespace with data from
	// Registry to give a more convenient interface for interacting with the
	// configuration as a whole.
	Namespace *Namespace

	// Registry is used alongside Namespace to determine the dynamic values
	// for objects declared in the namespace. Here at the root of a
	// configuration the Namespace/Registry distinction feels arbitrary, but
	// these two ideas are separated because child module calls using
	// "for_each" or "count" create a situation where there are potentially
	// many Registry instances associated with a single Namespace.
	//
	// Some methods of Config combine data from Registry with data from
	// Namespace to give a more convenient interface for interacting with the
	// configuration as a whole.
	Registry *Registry
}

// GenerateConfigSnapshot produces a configload.Snapshot containing the
// configuration source code defined in the reciever.
//
// A caller can then use configload.NewLoaderFromSnapshot as a first step
// toward loading the configuration to use when exercising other parts of
// Terraform.
func (c *Config) ConfigSnapshot() *configload.Snapshot {
	// Our config generator doesn't yet have any support for generating child
	// modules, so for now we're always just generating a single root module.
	// We'll need to rework this once c.Objects could potentially contain
	// objects representing child module calls.
	snap := &configload.Snapshot{
		Modules: map[string]*configload.SnapshotModule{},
	}
	c.populateSnapshotModules(addrs.RootModule, c.Objects, snap.Modules)
	return snap
}

// VariableValues returns values for input variables that the test harness
// must set when planning this configuration.
func (c *Config) VariableValues() map[string]cty.Value {
	ret := make(map[string]cty.Value, len(c.Registry.VariableValues))
	for addr, v := range c.Registry.VariableValues {
		ret[addr.Name] = v
	}
	return ret
}

// ConfigFile generates a configuration file containing the source code for
// each of the objects in the given slice.
func ConfigFile(objs []ConfigObject) []byte {
	f := hclwrite.NewEmptyFile()
	body := f.Body()
	for _, obj := range objs {
		obj.AppendConfig(body)
		body.AppendNewline()
	}
	return f.Bytes()
}

// CheckNewState asks each of the object instances in the configuration if the
// given new state matches the object's expected results, returning a nonzero
// number of errors if there are any problems.
func (c *Config) CheckNewState(prior, new *states.State) []error {
	var errs []error
	for _, inst := range c.ObjectInstances {
		for _, err := range inst.CheckState(prior, new) {
			errs = append(errs, fmt.Errorf("%s: %w", inst.DisplayName(), err))
		}
	}
	return errs
}

// populateSnapshotModules is the recursive main implementation of
// ConfigSnapshot, which walks the tree of modules implied by the given
// objects.
func (c *Config) populateSnapshotModules(addr addrs.Module, objs []ConfigObject, into map[string]*configload.SnapshotModule) {
	modKey := strings.Join(addr, ".")
	modPath := "./" + strings.Join(addr, "/")
	into[modKey] = &configload.SnapshotModule{
		Dir: modPath,
		Files: map[string][]byte{
			"test.tf": ConfigFile(objs),
		},
	}
	if !addr.IsRoot() {
		into[modKey].SourceAddr = "./" + addr[len(addr)-1]
	}
	// If any of the objects are *ConfigModuleCall instances then we need
	// to recursively generate those, too.
	for _, obj := range objs {
		if mc, ok := obj.(*ConfigModuleCall); ok {
			c.populateSnapshotModules(addr.Child(mc.Addr.Name), mc.Objects, into)
		}
	}
}
