package terraform

import (
	"bufio"
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/mitchellh/copystructure"
)

// DiffChangeType is an enum with the kind of changes a diff has planned.
type DiffChangeType byte

const (
	DiffInvalid DiffChangeType = iota
	DiffNone
	DiffCreate
	DiffUpdate
	DiffDestroy
	DiffDestroyCreate
)

// Diff trackes the changes that are necessary to apply a configuration
// to an existing infrastructure.
type Diff struct {
	// Modules contains all the modules that have a diff
	Modules []*ModuleDiff
}

// Prune cleans out unused structures in the diff without affecting
// the behavior of the diff at all.
//
// This is not safe to call concurrently. This is safe to call on a
// nil Diff.
func (d *Diff) Prune() {
	if d == nil {
		return
	}

	// Prune all empty modules
	newModules := make([]*ModuleDiff, 0, len(d.Modules))
	for _, m := range d.Modules {
		// If the module isn't empty, we keep it
		if !m.Empty() {
			newModules = append(newModules, m)
		}
	}
	if len(newModules) == 0 {
		newModules = nil
	}
	d.Modules = newModules
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
// This should be the preferred lookup mechanism as it allows for future
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
	if d == nil {
		return true
	}

	for _, m := range d.Modules {
		if !m.Empty() {
			return false
		}
	}

	return true
}

// Equal compares two diffs for exact equality.
//
// This is different from the Same comparison that is supported which
// checks for operation equality taking into account computed values. Equal
// instead checks for exact equality.
func (d *Diff) Equal(d2 *Diff) bool {
	// If one is nil, they must both be nil
	if d == nil || d2 == nil {
		return d == d2
	}

	// Sort the modules
	sort.Sort(moduleDiffSort(d.Modules))
	sort.Sort(moduleDiffSort(d2.Modules))

	// Copy since we have to modify the module destroy flag to false so
	// we don't compare that. TODO: delete this when we get rid of the
	// destroy flag on modules.
	dCopy := d.DeepCopy()
	d2Copy := d2.DeepCopy()
	for _, m := range dCopy.Modules {
		m.Destroy = false
	}
	for _, m := range d2Copy.Modules {
		m.Destroy = false
	}

	// Use DeepEqual
	return reflect.DeepEqual(dCopy, d2Copy)
}

// DeepCopy performs a deep copy of all parts of the Diff, making the
// resulting Diff safe to use without modifying this one.
func (d *Diff) DeepCopy() *Diff {
	copy, err := copystructure.Config{Lock: true}.Copy(d)
	if err != nil {
		panic(err)
	}

	return copy.(*Diff)
}

func (d *Diff) String() string {
	var buf bytes.Buffer

	keys := make([]string, 0, len(d.Modules))
	lookup := make(map[string]*ModuleDiff)
	for _, m := range d.Modules {
		key := fmt.Sprintf("module.%s", strings.Join(m.Path[1:], "."))
		keys = append(keys, key)
		lookup[key] = m
	}
	sort.Strings(keys)

	for _, key := range keys {
		m := lookup[key]
		mStr := m.String()

		// If we're the root module, we just write the output directly.
		if reflect.DeepEqual(m.Path, rootModulePath) {
			buf.WriteString(mStr + "\n")
			continue
		}

		buf.WriteString(fmt.Sprintf("%s:\n", key))

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
	Destroy   bool // Set only by the destroy plan
}

func (d *ModuleDiff) init() {
	if d.Resources == nil {
		d.Resources = make(map[string]*InstanceDiff)
	}
	for _, r := range d.Resources {
		r.init()
	}
}

// ChangeType returns the type of changes that the diff for this
// module includes.
//
// At a module level, this will only be DiffNone, DiffUpdate, DiffDestroy, or
// DiffCreate. If an instance within the module has a DiffDestroyCreate
// then this will register as a DiffCreate for a module.
func (d *ModuleDiff) ChangeType() DiffChangeType {
	result := DiffNone
	for _, r := range d.Resources {
		change := r.ChangeType()
		switch change {
		case DiffCreate, DiffDestroy:
			if result == DiffNone {
				result = change
			}
		case DiffDestroyCreate, DiffUpdate:
			result = DiffUpdate
		}
	}

	return result
}

// Empty returns true if the diff has no changes within this module.
func (d *ModuleDiff) Empty() bool {
	if d.Destroy {
		return false
	}

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

// Instances returns the instance diffs for the id given. This can return
// multiple instance diffs if there are counts within the resource.
func (d *ModuleDiff) Instances(id string) []*InstanceDiff {
	var result []*InstanceDiff
	for k, diff := range d.Resources {
		if k == id || strings.HasPrefix(k, id+".") {
			if !diff.Empty() {
				result = append(result, diff)
			}
		}
	}

	return result
}

// IsRoot says whether or not this module diff is for the root module.
func (d *ModuleDiff) IsRoot() bool {
	return reflect.DeepEqual(d.Path, rootModulePath)
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
		switch {
		case rdiff.RequiresNew() && (rdiff.GetDestroy() || rdiff.GetDestroyTainted()):
			crud = "DESTROY/CREATE"
		case rdiff.GetDestroy() || rdiff.GetDestroyDeposed():
			crud = "DESTROY"
		case rdiff.RequiresNew():
			crud = "CREATE"
		}

		extra := ""
		if !rdiff.GetDestroy() && rdiff.GetDestroyDeposed() {
			extra = " (deposed only)"
		}

		buf.WriteString(fmt.Sprintf(
			"%s: %s%s\n",
			crud,
			name,
			extra))

		keyLen := 0
		rdiffAttrs := rdiff.CopyAttributes()
		keys := make([]string, 0, len(rdiffAttrs))
		for key, _ := range rdiffAttrs {
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
			attrDiff, _ := rdiff.GetAttribute(attrK)

			v := attrDiff.New
			u := attrDiff.Old
			if attrDiff.NewComputed {
				v = "<computed>"
			}

			if attrDiff.Sensitive {
				u = "<sensitive>"
				v = "<sensitive>"
			}

			updateMsg := ""
			if attrDiff.RequiresNew {
				updateMsg = " (forces new resource)"
			} else if attrDiff.Sensitive {
				updateMsg = " (attribute changed)"
			}

			buf.WriteString(fmt.Sprintf(
				"  %s:%s %#v => %#v%s\n",
				attrK,
				strings.Repeat(" ", keyLen-len(attrK)),
				u,
				v,
				updateMsg))
		}
	}

	return buf.String()
}

// InstanceDiff is the diff of a resource from some state to another.
type InstanceDiff struct {
	mu             sync.Mutex
	Attributes     map[string]*ResourceAttrDiff
	Destroy        bool
	DestroyDeposed bool
	DestroyTainted bool
}

func (d *InstanceDiff) Lock()   { d.mu.Lock() }
func (d *InstanceDiff) Unlock() { d.mu.Unlock() }

// ResourceAttrDiff is the diff of a single attribute of a resource.
type ResourceAttrDiff struct {
	Old         string      // Old Value
	New         string      // New Value
	NewComputed bool        // True if new value is computed (unknown currently)
	NewRemoved  bool        // True if this attribute is being removed
	NewExtra    interface{} // Extra information for the provider
	RequiresNew bool        // True if change requires new resource
	Sensitive   bool        // True if the data should not be displayed in UI output
	Type        DiffAttrType
}

// Empty returns true if the diff for this attr is neutral
func (d *ResourceAttrDiff) Empty() bool {
	return d.Old == d.New && !d.NewComputed && !d.NewRemoved
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

func NewInstanceDiff() *InstanceDiff {
	return &InstanceDiff{Attributes: make(map[string]*ResourceAttrDiff)}
}

func (d *InstanceDiff) Copy() (*InstanceDiff, error) {
	if d == nil {
		return nil, nil
	}

	dCopy, err := copystructure.Config{Lock: true}.Copy(d)
	if err != nil {
		return nil, err
	}

	return dCopy.(*InstanceDiff), nil
}

// ChangeType returns the DiffChangeType represented by the diff
// for this single instance.
func (d *InstanceDiff) ChangeType() DiffChangeType {
	if d.Empty() {
		return DiffNone
	}

	if d.RequiresNew() && (d.GetDestroy() || d.GetDestroyTainted()) {
		return DiffDestroyCreate
	}

	if d.GetDestroy() || d.GetDestroyDeposed() {
		return DiffDestroy
	}

	if d.RequiresNew() {
		return DiffCreate
	}

	return DiffUpdate
}

// Empty returns true if this diff encapsulates no changes.
func (d *InstanceDiff) Empty() bool {
	if d == nil {
		return true
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	return !d.Destroy &&
		!d.DestroyTainted &&
		!d.DestroyDeposed &&
		len(d.Attributes) == 0
}

// Equal compares two diffs for exact equality.
//
// This is different from the Same comparison that is supported which
// checks for operation equality taking into account computed values. Equal
// instead checks for exact equality.
func (d *InstanceDiff) Equal(d2 *InstanceDiff) bool {
	// If one is nil, they must both be nil
	if d == nil || d2 == nil {
		return d == d2
	}

	// Use DeepEqual
	return reflect.DeepEqual(d, d2)
}

// DeepCopy performs a deep copy of all parts of the InstanceDiff
func (d *InstanceDiff) DeepCopy() *InstanceDiff {
	copy, err := copystructure.Config{Lock: true}.Copy(d)
	if err != nil {
		panic(err)
	}

	return copy.(*InstanceDiff)
}

func (d *InstanceDiff) GoString() string {
	return fmt.Sprintf("*%#v", InstanceDiff{
		Attributes:     d.Attributes,
		Destroy:        d.Destroy,
		DestroyTainted: d.DestroyTainted,
		DestroyDeposed: d.DestroyDeposed,
	})
}

// RequiresNew returns true if the diff requires the creation of a new
// resource (implying the destruction of the old).
func (d *InstanceDiff) RequiresNew() bool {
	if d == nil {
		return false
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	return d.requiresNew()
}

func (d *InstanceDiff) requiresNew() bool {
	if d == nil {
		return false
	}

	if d.DestroyTainted {
		return true
	}

	for _, rd := range d.Attributes {
		if rd != nil && rd.RequiresNew {
			return true
		}
	}

	return false
}

func (d *InstanceDiff) GetDestroyDeposed() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.DestroyDeposed
}

func (d *InstanceDiff) SetDestroyDeposed(b bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.DestroyDeposed = b
}

// These methods are properly locked, for use outside other InstanceDiff
// methods but everywhere else within in the terraform package.
// TODO refactor the locking scheme
func (d *InstanceDiff) SetTainted(b bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.DestroyTainted = b
}

func (d *InstanceDiff) GetDestroyTainted() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.DestroyTainted
}

func (d *InstanceDiff) SetDestroy(b bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.Destroy = b
}

func (d *InstanceDiff) GetDestroy() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.Destroy
}

func (d *InstanceDiff) SetAttribute(key string, attr *ResourceAttrDiff) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.Attributes[key] = attr
}

func (d *InstanceDiff) DelAttribute(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.Attributes, key)
}

func (d *InstanceDiff) GetAttribute(key string) (*ResourceAttrDiff, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	attr, ok := d.Attributes[key]
	return attr, ok
}
func (d *InstanceDiff) GetAttributesLen() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return len(d.Attributes)
}

// Safely copies the Attributes map
func (d *InstanceDiff) CopyAttributes() map[string]*ResourceAttrDiff {
	d.mu.Lock()
	defer d.mu.Unlock()

	attrs := make(map[string]*ResourceAttrDiff)
	for k, v := range d.Attributes {
		attrs[k] = v
	}

	return attrs
}

// Same checks whether or not two InstanceDiff's are the "same". When
// we say "same", it is not necessarily exactly equal. Instead, it is
// just checking that the same attributes are changing, a destroy
// isn't suddenly happening, etc.
func (d *InstanceDiff) Same(d2 *InstanceDiff) (bool, string) {
	// we can safely compare the pointers without a lock
	switch {
	case d == nil && d2 == nil:
		return true, ""
	case d == nil || d2 == nil:
		return false, "one nil"
	case d == d2:
		return true, ""
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// If we're going from requiring new to NOT requiring new, then we have
	// to see if all required news were computed. If so, it is allowed since
	// computed may also mean "same value and therefore not new".
	oldNew := d.requiresNew()
	newNew := d2.RequiresNew()
	if oldNew && !newNew {
		oldNew = false
		for _, rd := range d.Attributes {
			// If the field is requires new and NOT computed, then what
			// we have is a diff mismatch for sure. We set that the old
			// diff does REQUIRE a ForceNew.
			if rd != nil && rd.RequiresNew && !rd.NewComputed {
				oldNew = true
				break
			}
		}
	}

	if oldNew != newNew {
		return false, fmt.Sprintf(
			"diff RequiresNew; old: %t, new: %t", oldNew, newNew)
	}

	// Verify that destroy matches. The second boolean here allows us to
	// have mismatching Destroy if we're moving from RequiresNew true
	// to false above. Therefore, the second boolean will only pass if
	// we're moving from Destroy: true to false as well.
	if d.Destroy != d2.GetDestroy() && d.requiresNew() == oldNew {
		return false, fmt.Sprintf(
			"diff: Destroy; old: %t, new: %t", d.Destroy, d2.GetDestroy())
	}

	// Go through the old diff and make sure the new diff has all the
	// same attributes. To start, build up the check map to be all the keys.
	checkOld := make(map[string]struct{})
	checkNew := make(map[string]struct{})
	for k, _ := range d.Attributes {
		checkOld[k] = struct{}{}
	}
	for k, _ := range d2.CopyAttributes() {
		checkNew[k] = struct{}{}
	}

	// Make an ordered list so we are sure the approximated hashes are left
	// to process at the end of the loop
	keys := make([]string, 0, len(d.Attributes))
	for k, _ := range d.Attributes {
		keys = append(keys, k)
	}
	sort.StringSlice(keys).Sort()

	for _, k := range keys {
		diffOld := d.Attributes[k]

		if _, ok := checkOld[k]; !ok {
			// We're not checking this key for whatever reason (see where
			// check is modified).
			continue
		}

		// Remove this key since we'll never hit it again
		delete(checkOld, k)
		delete(checkNew, k)

		_, ok := d2.GetAttribute(k)
		if !ok {
			// If there's no new attribute, and the old diff expected the attribute
			// to be removed, that's just fine.
			if diffOld.NewRemoved {
				continue
			}

			// If the last diff was a computed value then the absense of
			// that value is allowed since it may mean the value ended up
			// being the same.
			if diffOld.NewComputed {
				ok = true
			}

			// No exact match, but maybe this is a set containing computed
			// values. So check if there is an approximate hash in the key
			// and if so, try to match the key.
			if strings.Contains(k, "~") {
				parts := strings.Split(k, ".")
				parts2 := append([]string(nil), parts...)

				re := regexp.MustCompile(`^~\d+$`)
				for i, part := range parts {
					if re.MatchString(part) {
						// we're going to consider this the base of a
						// computed hash, and remove all longer matching fields
						ok = true

						parts2[i] = `\d+`
						parts2 = parts2[:i+1]
						break
					}
				}

				re, err := regexp.Compile("^" + strings.Join(parts2, `\.`))
				if err != nil {
					return false, fmt.Sprintf("regexp failed to compile; err: %#v", err)
				}

				for k2, _ := range checkNew {
					if re.MatchString(k2) {
						delete(checkNew, k2)
					}
				}
			}

			// This is a little tricky, but when a diff contains a computed
			// list, set, or map that can only be interpolated after the apply
			// command has created the dependent resources, it could turn out
			// that the result is actually the same as the existing state which
			// would remove the key from the diff.
			if diffOld.NewComputed && (strings.HasSuffix(k, ".#") || strings.HasSuffix(k, ".%")) {
				ok = true
			}

			// Similarly, in a RequiresNew scenario, a list that shows up in the plan
			// diff can disappear from the apply diff, which is calculated from an
			// empty state.
			if d.requiresNew() && (strings.HasSuffix(k, ".#") || strings.HasSuffix(k, ".%")) {
				ok = true
			}

			if !ok {
				return false, fmt.Sprintf("attribute mismatch: %s", k)
			}
		}

		// search for the suffix of the base of a [computed] map, list or set.
		multiVal := regexp.MustCompile(`\.(#|~#|%)$`)
		match := multiVal.FindStringSubmatch(k)

		if diffOld.NewComputed && len(match) == 2 {
			matchLen := len(match[1])

			// This is a computed list, set, or map, so remove any keys with
			// this prefix from the check list.
			kprefix := k[:len(k)-matchLen]
			for k2, _ := range checkOld {
				if strings.HasPrefix(k2, kprefix) {
					delete(checkOld, k2)
				}
			}
			for k2, _ := range checkNew {
				if strings.HasPrefix(k2, kprefix) {
					delete(checkNew, k2)
				}
			}
		}

		// TODO: check for the same value if not computed
	}

	// Check for leftover attributes
	if len(checkNew) > 0 {
		extras := make([]string, 0, len(checkNew))
		for attr, _ := range checkNew {
			extras = append(extras, attr)
		}
		return false,
			fmt.Sprintf("extra attributes: %s", strings.Join(extras, ", "))
	}

	return true, ""
}

// moduleDiffSort implements sort.Interface to sort module diffs by path.
type moduleDiffSort []*ModuleDiff

func (s moduleDiffSort) Len() int      { return len(s) }
func (s moduleDiffSort) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s moduleDiffSort) Less(i, j int) bool {
	a := s[i]
	b := s[j]

	// If the lengths are different, then the shorter one always wins
	if len(a.Path) != len(b.Path) {
		return len(a.Path) < len(b.Path)
	}

	// Otherwise, compare lexically
	return strings.Join(a.Path, ".") < strings.Join(b.Path, ".")
}
