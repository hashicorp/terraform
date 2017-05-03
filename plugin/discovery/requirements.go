package discovery

// PluginRequirements describes a set of plugins (assumed to be of a consistent
// kind) that are required to exist and have versions within the given
// corresponding sets.
//
// PluginRequirements is a map from plugin name to Constraints.
type PluginRequirements map[string]Constraints

// Merge takes the contents of the receiver and the other given requirements
// object and merges them together into a single requirements structure
// that satisfies both sets of requirements.
func (r PluginRequirements) Merge(other PluginRequirements) PluginRequirements {
	ret := make(PluginRequirements)
	for n, vs := range r {
		ret[n] = vs
	}
	for n, vs := range other {
		if existing, exists := ret[n]; exists {
			ret[n] = existing.Intersection(vs)
		} else {
			ret[n] = vs
		}
	}
	return ret
}
