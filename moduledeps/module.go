package moduledeps

import (
	"sort"
	"strings"

	"github.com/hashicorp/terraform/plugin/discovery"
)

// Module represents the dependencies of a single module, as well being
// a node in a tree of such structures representing the dependencies of
// an entire configuration.
type Module struct {
	Name      string
	Providers Providers
	Children  []*Module
}

// WalkFunc is a callback type for use with Module.WalkTree
type WalkFunc func(path []string, parent *Module, current *Module) error

// WalkTree calls the given callback once for the receiver and then
// once for each descendent, in an order such that parents are called
// before their children and siblings are called in the order they
// appear in the Children slice.
//
// When calling the callback, parent will be nil for the first call
// for the receiving module, and then set to the direct parent of
// each module for the subsequent calls.
//
// The path given to the callback is valid only until the callback
// returns, after which it will be mutated and reused. Callbacks must
// therefore copy the path slice if they wish to retain it.
//
// If the given callback returns an error, the walk will be aborted at
// that point and that error returned to the caller.
//
// This function is not thread-safe for concurrent modifications of the
// data structure, so it's the caller's responsibility to arrange for that
// should it be needed.
//
// It is safe for a callback to modify the descendents of the "current"
// module, including the ordering of the Children slice itself, but the
// callback MUST NOT modify the parent module.
func (m *Module) WalkTree(cb WalkFunc) error {
	return walkModuleTree(make([]string, 0, 1), nil, m, cb)
}

func walkModuleTree(path []string, parent *Module, current *Module, cb WalkFunc) error {
	path = append(path, current.Name)
	err := cb(path, parent, current)
	if err != nil {
		return err
	}

	for _, child := range current.Children {
		err := walkModuleTree(path, current, child, cb)
		if err != nil {
			return err
		}
	}
	return nil
}

// SortChildren sorts the Children slice into lexicographic order by
// name, in-place.
//
// This is primarily useful prior to calling WalkTree so that the walk
// will proceed in a consistent order.
func (m *Module) SortChildren() {
	sort.Sort(sortModules{m.Children})
}

// SortDescendents is a convenience wrapper for calling SortChildren on
// the receiver and all of its descendent modules.
func (m *Module) SortDescendents() {
	m.WalkTree(func(path []string, parent *Module, current *Module) error {
		current.SortChildren()
		return nil
	})
}

type sortModules struct {
	modules []*Module
}

func (s sortModules) Len() int {
	return len(s.modules)
}

func (s sortModules) Less(i, j int) bool {
	cmp := strings.Compare(s.modules[i].Name, s.modules[j].Name)
	return cmp < 0
}

func (s sortModules) Swap(i, j int) {
	s.modules[i], s.modules[j] = s.modules[j], s.modules[i]
}

// PluginRequirements produces a PluginRequirements structure that can
// be used with discovery.PluginMetaSet.ConstrainVersions to identify
// suitable plugins to satisfy the module's provider dependencies.
//
// This method only considers the direct requirements of the receiver.
// Use AllPluginRequirements to flatten the dependencies for the
// entire tree of modules.
//
// Requirements returned by this method include only version constraints,
// and apply no particular SHA256 hash constraint.
func (m *Module) PluginRequirements() discovery.PluginRequirements {
	ret := make(discovery.PluginRequirements)
	for inst, dep := range m.Providers {
		// m.Providers is keyed on provider names, such as "aws.foo".
		// a PluginRequirements wants keys to be provider *types*, such
		// as "aws". If there are multiple aliases for the same
		// provider then we will flatten them into a single requirement
		// by combining their constraint sets.
		pty := inst.Type()
		if existing, exists := ret[pty]; exists {
			ret[pty].Versions = existing.Versions.Append(dep.Constraints)
		} else {
			ret[pty] = &discovery.PluginConstraints{
				Versions: dep.Constraints,
			}
		}
	}
	return ret
}

// AllPluginRequirements calls PluginRequirements for the receiver and all
// of its descendents, and merges the result into a single PluginRequirements
// structure that would satisfy all of the modules together.
//
// Requirements returned by this method include only version constraints,
// and apply no particular SHA256 hash constraint.
func (m *Module) AllPluginRequirements() discovery.PluginRequirements {
	var ret discovery.PluginRequirements
	m.WalkTree(func(path []string, parent *Module, current *Module) error {
		ret = ret.Merge(current.PluginRequirements())
		return nil
	})
	return ret
}

// Equal returns true if the receiver is the root of an identical tree
// to the other given Module. This is a deep comparison that considers
// the equality of all downstream modules too.
//
// The children are considered to be ordered, so callers may wish to use
// SortDescendents first to normalize the order of the slices of child nodes.
//
// The implementation of this function is not optimized since it is provided
// primarily for use in tests.
func (m *Module) Equal(other *Module) bool {
	// take care of nils first
	if m == nil && other == nil {
		return true
	} else if (m == nil && other != nil) || (m != nil && other == nil) {
		return false
	}

	if m.Name != other.Name {
		return false
	}

	if len(m.Providers) != len(other.Providers) {
		return false
	}
	if len(m.Children) != len(other.Children) {
		return false
	}

	// Can't use reflect.DeepEqual on this provider structure because
	// the nested Constraints objects contain function pointers that
	// never compare as equal. So we'll need to walk it the long way.
	for inst, dep := range m.Providers {
		if _, exists := other.Providers[inst]; !exists {
			return false
		}

		if dep.Reason != other.Providers[inst].Reason {
			return false
		}

		// Constraints are not too easy to compare robustly, so
		// we'll just use their string representations as a proxy
		// for now.
		if dep.Constraints.String() != other.Providers[inst].Constraints.String() {
			return false
		}
	}

	// Above we already checked that we have the same number of children
	// in each module, so now we just need to check that they are
	// recursively equal.
	for i := range m.Children {
		if !m.Children[i].Equal(other.Children[i]) {
			return false
		}
	}

	// If we fall out here then they are equal
	return true
}
