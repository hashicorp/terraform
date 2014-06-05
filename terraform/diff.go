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

// ResourceDiff is the diff of a resource from some state to another.
type ResourceDiff struct {
	Attributes map[string]*ResourceAttrDiff
}

// ResourceAttrDiff is the diff of a single attribute of a resource.
type ResourceAttrDiff struct {
	Old         string // Old Value
	New         string // New Value
	NewComputed bool   // True if new value is computed (unknown currently)
	RequiresNew bool   // True if change requires new resource
}
