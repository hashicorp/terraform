package terraform

// Diff tracks the differences between resources to apply.
type Diff struct {
	resources map[string]map[string]*resourceDiff
}

// resourceDiff is the diff of a single attribute of a resource.
//
// This tracks the old value, the new value, and whether the change of this
// value actually requires an entirely new resource.
type resourceDiff struct {
	Old         string
	New         string
	RequiresNew bool
}
