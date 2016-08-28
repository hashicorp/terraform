package terraform

import (
	"fmt"
	"reflect"
	"strings"
)

// Deferrals represents a set of nodes whose evaluation is deferred to a later
// run of Terraform.
//
// This set is populated by both the "validate" and "plan" walks, as we discover
// situations where the configuration is not yet sufficiently complete to form
// a complete graph.
//
// The deferral set is used during graph construction to suppress any deferred
// nodes along with any other nodes that depend on them. Thus nodes deferred during
// "validate" are skipped during all subsequent phases, and nodes deferred during
// "plan" are skipped during apply.
type Deferrals struct {
	Modules []*ModuleDeferrals
}

func NewDeferrals() *Deferrals {
	return &Deferrals{
		Modules: []*ModuleDeferrals{},
	}
}

func (d *Deferrals) ModuleByPath(path []string) *ModuleDeferrals {
	if d == nil {
		return nil
	}
	for _, mod := range d.Modules {
		if mod.Path == nil {
			panic("missing module path")
		}
		if reflect.DeepEqual(mod.Path, path) {
			return mod
		}
	}
	return nil
}

func (d *Deferrals) AddModule(path []string) *ModuleDeferrals {
	m := &ModuleDeferrals{
		Path: path,

		Providers: map[string]string{},
		Resources: map[string]string{},
	}
	d.Modules = append(d.Modules, m)
	return m
}

func (d *Deferrals) Empty() bool {
	if d == nil {
		return true
	}

	for _, md := range d.Modules {
		if len(md.Providers)+len(md.Resources) != 0 {
			return false
		}
	}
	return true
}

// ModuleDeferrals is the set of deferrals for a particular module. It represents
// part of a full set of deferred nodes within a Deferrals instance.
type ModuleDeferrals struct {
	Path []string

	// This "set" is actually represented as a map for each deferrable node type,
	// whose key is a type-specific identifier and whose value is an English-language
	// reason for why the node is deferred.

	Providers map[string]string
	Resources map[string]string
}

func (md *ModuleDeferrals) ModuleDisplayPrefix() string {
	if len(md.Path) == 1 && md.Path[0] == "root" {
		return ""
	}

	return fmt.Sprintf("module.%s.", strings.Join(md.Path[1:], ".module."))
}

func (md *ModuleDeferrals) DeferProvider(name string, reason string) {
	md.Providers[name] = reason
}

func (md *ModuleDeferrals) DeferResource(name string, reason string) {
	md.Resources[name] = reason
}

func (md *ModuleDeferrals) ProviderIsDeferred(name string) bool {
	_, deferred := md.Providers[name]
	return deferred
}
