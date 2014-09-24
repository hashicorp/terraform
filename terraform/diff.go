package terraform

import (
	"bufio"
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Diff trackes the changes that are necessary to apply a configuration
// to an existing infrastructure.
type Diff struct {
	// Modules contains all the modules that have a diff
	Modules []*ModuleDiff
}

// AddModule adds the module with the given path to the diff.
//
// This should be the preferred method to add module diffs since it
// allows us to optimize lookups later as well as control sorting.
func (d *Diff) AddModule(path []string) *ModuleDiff {
	m := &ModuleDiff{Path: path}
	m.init()
	d.Modules = append(d.Modules, m)
	return m
}

// ModuleByPath is used to lookup the module diff for the given path.
// This should be the prefered lookup mechanism as it allows for future
// lookup optimizations.
func (d *Diff) ModuleByPath(path []string) *ModuleDiff {
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

// RootModule returns the ModuleState for the root module
func (d *Diff) RootModule() *ModuleDiff {
	root := d.ModuleByPath(rootModulePath)
	if root == nil {
		panic("missing root module")
	}
	return root
}

// Empty returns true if the diff has no changes.
func (d *Diff) Empty() bool {
	for _, m := range d.Modules {
		if !m.Empty() {
			return false
		}
	}

	return true
}

func (d *Diff) String() string {
	var buf bytes.Buffer
	for _, m := range d.Modules {
		mStr := m.String()

		// If we're the root module, we just write the output directly.
		if reflect.DeepEqual(m.Path, rootModulePath) {
			buf.WriteString(mStr + "\n")
			continue
		}

		buf.WriteString(fmt.Sprintf("module.%s:\n", strings.Join(m.Path[1:], ".")))

		s := bufio.NewScanner(strings.NewReader(mStr))
		for s.Scan() {
			buf.WriteString(fmt.Sprintf("  %s\n", s.Text()))
		}
	}

	return strings.TrimSpace(buf.String())
}

func (d *Diff) init() {
	if d.Modules == nil {
		rootDiff := &ModuleDiff{Path: rootModulePath}
		d.Modules = []*ModuleDiff{rootDiff}
	}
	for _, m := range d.Modules {
		m.init()
	}
}

// ModuleDiff tracks the differences between resources to apply within
// a single module.
type ModuleDiff struct {
	Path      []string
	Resources map[string]*InstanceDiff
}

func (d *ModuleDiff) init() {
	if d.Resources == nil {
		d.Resources = make(map[string]*InstanceDiff)
	}
	for _, r := range d.Resources {
		r.init()
	}
}

// Empty returns true if the diff has no changes within this module.
func (d *ModuleDiff) Empty() bool {
	if len(d.Resources) == 0 {
		return true
	}

	for _, rd := range d.Resources {
		if !rd.Empty() {
			return false
		}
	}

	return true
}

// String outputs the diff in a long but command-line friendly output
// format that users can read to quickly inspect a diff.
func (d *ModuleDiff) String() string {
	var buf bytes.Buffer

	names := make([]string, 0, len(d.Resources))
	for name, _ := range d.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		rdiff := d.Resources[name]

		crud := "UPDATE"
		if rdiff.RequiresNew() && (rdiff.Destroy || rdiff.DestroyTainted) {
			crud = "DESTROY/CREATE"
		} else if rdiff.Destroy {
			crud = "DESTROY"
		} else if rdiff.RequiresNew() {
			crud = "CREATE"
		}

		buf.WriteString(fmt.Sprintf(
			"%s: %s\n",
			crud,
			name))

		keyLen := 0
		keys := make([]string, 0, len(rdiff.Attributes))
		for key, _ := range rdiff.Attributes {
			if key == "id" {
				continue
			}

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

			newResource := ""
			if attrDiff.RequiresNew {
				newResource = " (forces new resource)"
			}

			buf.WriteString(fmt.Sprintf(
				"  %s:%s %#v => %#v%s\n",
				attrK,
				strings.Repeat(" ", keyLen-len(attrK)),
				attrDiff.Old,
				v,
				newResource))
		}
	}

	return buf.String()
}

// InstanceDiff is the diff of a resource from some state to another.
type InstanceDiff struct {
	Attributes     map[string]*ResourceAttrDiff
	Destroy        bool
	DestroyTainted bool
}

// ResourceAttrDiff is the diff of a single attribute of a resource.
type ResourceAttrDiff struct {
	Old         string      // Old Value
	New         string      // New Value
	NewComputed bool        // True if new value is computed (unknown currently)
	NewRemoved  bool        // True if this attribute is being removed
	NewExtra    interface{} // Extra information for the provider
	RequiresNew bool        // True if change requires new resource
	Type        DiffAttrType
}

func (d *ResourceAttrDiff) GoString() string {
	return fmt.Sprintf("*%#v", *d)
}

// DiffAttrType is an enum type that says whether a resource attribute
// diff is an input attribute (comes from the configuration) or an
// output attribute (comes as a result of applying the configuration). An
// example input would be "ami" for AWS and an example output would be
// "private_ip".
type DiffAttrType byte

const (
	DiffAttrUnknown DiffAttrType = iota
	DiffAttrInput
	DiffAttrOutput
)

func (d *InstanceDiff) init() {
	if d.Attributes == nil {
		d.Attributes = make(map[string]*ResourceAttrDiff)
	}
}

// Empty returns true if this diff encapsulates no changes.
func (d *InstanceDiff) Empty() bool {
	if d == nil {
		return true
	}

	return !d.Destroy && len(d.Attributes) == 0
}

// RequiresNew returns true if the diff requires the creation of a new
// resource (implying the destruction of the old).
func (d *InstanceDiff) RequiresNew() bool {
	if d == nil {
		return false
	}

	for _, rd := range d.Attributes {
		if rd != nil && rd.RequiresNew {
			return true
		}
	}

	return false
}

// Same checks whether or not to InstanceDiff are the "same." When
// we say "same", it is not necessarily exactly equal. Instead, it is
// just checking that the same attributes are changing, a destroy
// isn't suddenly happening, etc.
func (d *InstanceDiff) Same(d2 *InstanceDiff) bool {
	if d == nil && d2 == nil {
		return true
	} else if d == nil || d2 == nil {
		return false
	}

	if d.Destroy != d2.Destroy {
		return false
	}
	if d.RequiresNew() != d2.RequiresNew() {
		return false
	}
	if len(d.Attributes) != len(d2.Attributes) {
		return false
	}

	ks := make(map[string]struct{})
	for k, _ := range d.Attributes {
		ks[k] = struct{}{}
	}
	for k, _ := range d2.Attributes {
		delete(ks, k)
	}

	if len(ks) > 0 {
		return false
	}

	return true
}
