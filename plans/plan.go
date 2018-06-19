package plans

// Plan is the top-level type representing a planned set of changes.
//
// A plan is a summary of the set of changes required to move from a current
// state to a goal state derived from configuration. The described changes
// are not applied directly, but contain an approximation of the final
// result that will be completed during apply by resolving any values that
// cannot be predicted.
//
// A plan must always be accompanied by the state and configuration it was
// built from, since the plan does not itself include all of the information
// required to make the changes indicated.
type Plan struct {
	VariableValues  map[string]DynamicValue
	Changes         *Changes
	ProviderSHA256s map[string][]byte
}
