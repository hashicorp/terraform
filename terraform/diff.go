package terraform

import (
	"sync"
)

// Diff tracks the differences between resources to apply.
type Diff struct {
	Resources map[string]map[string]*ResourceAttrDiff
	once      sync.Once
}

func (d *Diff) init() {
	d.once.Do(func() {
		d.Resources = make(map[string]map[string]*ResourceAttrDiff)
	})
}

// ResourceAttrDiff is the diff of a single attribute of a resource.
//
// This tracks the old value, the new value, and whether the change of this
// value actually requires an entirely new resource.
type ResourceAttrDiff struct {
	Old         string
	New         string
	RequiresNew bool
}
