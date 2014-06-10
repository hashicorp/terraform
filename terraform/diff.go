package terraform

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Diff tracks the differences between resources to apply.
type Diff struct {
	Resources map[string]*ResourceDiff
	once      sync.Once
}

func (d *Diff) init() {
	d.once.Do(func() {
		if d.Resources == nil {
			d.Resources = make(map[string]*ResourceDiff)
		}
	})
}

// String outputs the diff in a long but command-line friendly output
// format that users can read to quickly inspect a diff.
func (d *Diff) String() string {
	var buf bytes.Buffer

	names := make([]string, 0, len(d.Resources))
	for name, _ := range d.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		rdiff := d.Resources[name]

		buf.WriteString(name + "\n")

		keyLen := 0
		keys := make([]string, 0, len(rdiff.Attributes))
		for key, _ := range rdiff.Attributes {
			keys = append(keys, key)
			if len(key) > keyLen {
				keyLen = len(key)
			}
		}
		sort.Strings(keys)

		for _, attrK := range keys {
			attrDiff := rdiff.Attributes[attrK]

			v := attrDiff.New
			if attrDiff.NewComputed {
				v = "<computed>"
			}

			buf.WriteString(fmt.Sprintf(
				"  %s:%s %#v => %#v\n",
				attrK,
				strings.Repeat(" ", keyLen-len(attrK)),
				attrDiff.Old,
				v))
		}
	}

	return buf.String()
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
