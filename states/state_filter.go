package states

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/addrs"
)

// Filter is responsible for filtering and searching a state.
//
// This is a separate struct from State rather than a method on State
// because Filter might create sidecar data structures to optimize
// filtering on the state.
//
// If you change the State, the filter created is invalid and either
// Reset should be called or a new one should be allocated. Filter
// will not watch State for changes and do this for you. If you filter after
// changing the State without calling Reset, the behavior is not defined.
type Filter struct {
	State *State
}

// Filter takes the addresses specified by fs and finds all the matches.
// The values of fs are resource addressing syntax that can be parsed by
// ParseResourceAddress.
func (f *Filter) Filter(fs ...string) ([]*FilterResult, error) {
	// Parse all the addresses
	as := make([]addrs.Targetable, len(fs))
	for i, v := range fs {
		if addr, diags := addrs.ParseModuleInstanceStr(v); !diags.HasErrors() {
			as[i] = addr
			continue
		}
		if addr, diags := addrs.ParseAbsResourceStr(v); !diags.HasErrors() {
			as[i] = addr
			continue
		}
		if addr, diags := addrs.ParseAbsResourceInstanceStr(v); !diags.HasErrors() {
			as[i] = addr
			continue
		}
		return nil, fmt.Errorf("Error parsing address '%s'", v)
	}

	// If we weren't given any filters, then we list all
	if len(fs) == 0 {
		as = append(as, addrs.Targetable(nil))
	}

	// Filter each of the address. We keep track of this in a map to
	// strip duplicates.
	resultSet := make(map[string]*FilterResult)
	for _, addr := range as {
		for _, v := range f.filterSingle(addr) {
			resultSet[v.String()] = v
		}
	}

	// Make the result list
	results := make([]*FilterResult, 0, len(resultSet))
	for _, v := range resultSet {
		results = append(results, v)
	}

	// Sort them and return
	sort.Sort(FilterResultSlice(results))
	return results, nil
}

func (f *Filter) filterSingle(addr addrs.Targetable) []*FilterResult {
	// The slice to keep track of results
	var results []*FilterResult

	// Check if we received a module instance address that
	// should be used as module filter, and if not set the
	// filter to the root module instance.
	filter, ok := addr.(addrs.ModuleInstance)
	if !ok {
		filter = addrs.RootModuleInstance
	}

	// Go through modules first.
	modules := make([]*Module, 0, len(f.State.Modules))
	for _, m := range f.State.Modules {
		if filter.IsRoot() || filter.Equal(m.Addr) || filter.IsAncestor(m.Addr) {
			modules = append(modules, m)

			// Only add the module to the results if we searched
			// for a non-root module and found a (partial) match.
			if (addr == nil && !m.Addr.IsRoot()) ||
				(!filter.IsRoot() && (filter.Equal(m.Addr) || filter.IsAncestor(m.Addr))) {
				results = append(results, &FilterResult{
					Address: m.Addr,
					Value:   m,
				})
			}
		}
	}

	// With the modules set, go through all the resources within
	// the modules to find relevant resources.
	for _, m := range modules {
		for _, rs := range m.Resources {
			if f.relevant(addr, rs.Addr.Absolute(m.Addr), addrs.NoKey) {
				results = append(results, &FilterResult{
					Address: rs.Addr.Absolute(m.Addr),
					Value:   rs,
				})
			}

			for key, is := range rs.Instances {
				if f.relevant(addr, rs.Addr.Absolute(m.Addr), key) {
					results = append(results, &FilterResult{
						Address: rs.Addr.Absolute(m.Addr).Instance(key),
						Value:   is,
					})
				}
			}
		}
	}

	return results
}

func (f *Filter) relevant(filter addrs.Targetable, rs addrs.AbsResource, key addrs.InstanceKey) bool {
	switch filter := filter.(type) {
	case addrs.AbsResource:
		if filter.Module != nil {
			return filter.Equal(rs)
		}
		return filter.Resource.Equal(rs.Resource)
	case addrs.AbsResourceInstance:
		if filter.Module != nil {
			return filter.Equal(rs.Instance(key))
		}
		return filter.Resource.Equal(rs.Resource.Instance(key))
	default:
		return true
	}
}

// FilterResult is a single result from a filter operation. Filter can
// match multiple things within a state (curently modules and resources).
type FilterResult struct {
	// Address is the address that can be used to reference this exact result.
	Address addrs.Targetable

	// Value is the actual value. This must be type switched on. It can be
	// any either a `Module` or `ResourceInstance`.
	Value interface{}
}

func (r *FilterResult) String() string {
	return fmt.Sprintf("%T: %s", r.Value, r.Address)
}

func (r *FilterResult) sortedType() int {
	switch r.Value.(type) {
	case *Module:
		return 0
	case *Resource:
		return 1
	case *ResourceInstance:
		return 2
	default:
		return 50
	}
}

// FilterResultSlice is a slice of results that implements
// sort.Interface. The sorting goal is what is most appealing to
// human output.
type FilterResultSlice []*FilterResult

func (s FilterResultSlice) Len() int      { return len(s) }
func (s FilterResultSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s FilterResultSlice) Less(i, j int) bool {
	a, b := s[i], s[j]

	// If the addresses are different it is just lexographic sorting
	if a.Address.String() != b.Address.String() {
		return a.Address.String() < b.Address.String()
	}

	// Addresses are the same, which means it matters on the type
	return a.sortedType() < b.sortedType()
}
