package terraform

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"

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

	// DiffRefresh is only used in the UI for displaying diffs.
	// Managed resource reads never appear in plan, and when data source
	// reads appear they are represented as DiffCreate in core before
	// transforming to DiffRefresh in the UI layer.
	DiffRefresh // TODO: Actually use DiffRefresh in core too, for less confusion
)

// multiVal matches the index key to a flatmapped set, list or map
var multiVal = regexp.MustCompile(`\.(#|%)$`)

// Diff tracks the changes that are necessary to apply a configuration
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
func (d *Diff) AddModule(path addrs.ModuleInstance) *ModuleDiff {
	// Lower the new-style address into a legacy-style address.
	// This requires that none of the steps have instance keys, which is
	// true for all addresses at the time of implementing this because
	// "count" and "for_each" are not yet implemented for modules.
	legacyPath := make([]string, len(path))
	for i, step := range path {
		if step.InstanceKey != addrs.NoKey {
			// FIXME: Once the rest of Terraform is ready to use count and
			// for_each, remove all of this and just write the addrs.ModuleInstance
			// value itself into the ModuleState.
			panic("diff cannot represent modules with count or for_each keys")
		}

		legacyPath[i] = step.Name
	}

	m := &ModuleDiff{Path: legacyPath}
	m.init()
	d.Modules = append(d.Modules, m)
	return m
}

// ModuleByPath is used to lookup the module diff for the given path.
// This should be the preferred lookup mechanism as it allows for future
// lookup optimizations.
func (d *Diff) ModuleByPath(path addrs.ModuleInstance) *ModuleDiff {
	if d == nil {
		return nil
	}
	for _, mod := range d.Modules {
		if mod.Path == nil {
			panic("missing module path")
		}
		modPath := normalizeModulePath(mod.Path)
		if modPath.String() == path.String() {
			return mod
		}
	}
	return nil
}

// RootModule returns the ModuleState for the root module
func (d *Diff) RootModule() *ModuleDiff {
	root := d.ModuleByPath(addrs.RootModuleInstance)
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
		addr := normalizeModulePath(m.Path)
		key := addr.String()
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

	// Meta is a simple K/V map that is stored in a diff and persisted to
	// plans but otherwise is completely ignored by Terraform core. It is
	// meant to be used for additional data a resource may want to pass through.
	// The value here must only contain Go primitives and collections.
	Meta map[string]interface{}
}

func (d *InstanceDiff) Lock()   { d.mu.Lock() }
func (d *InstanceDiff) Unlock() { d.mu.Unlock() }

// ApplyToValue merges the receiver into the given base value, returning a
// new value that incorporates the planned changes. The given value must
// conform to the given schema, or this method will panic.
//
// This method is intended for shimming old subsystems that still use this
// legacy diff type to work with the new-style types.
func (d *InstanceDiff) ApplyToValue(base cty.Value, schema *configschema.Block) (cty.Value, error) {
	// Create an InstanceState attributes from our existing state.
	// We can use this to more easily apply the diff changes.
	attrs := hcl2shim.FlatmapValueFromHCL2(base)
	applied, err := d.Apply(attrs, schema)
	if err != nil {
		return base, err
	}

	val, err := hcl2shim.HCL2ValueFromFlatmap(applied, schema.ImpliedType())
	if err != nil {
		return base, err
	}

	return schema.CoerceValue(val)
}

// Apply applies the diff to the provided flatmapped attributes,
// returning the new instance attributes.
//
// This method is intended for shimming old subsystems that still use this
// legacy diff type to work with the new-style types.
func (d *InstanceDiff) Apply(attrs map[string]string, schema *configschema.Block) (map[string]string, error) {
	// We always build a new value here, even if the given diff is "empty",
	// because we might be planning to create a new instance that happens
	// to have no attributes set, and so we want to produce an empty object
	// rather than just echoing back the null old value.
	if attrs == nil {
		attrs = map[string]string{}
	}

	// Rather applying the diff to mutate the attrs, we'll copy new values into
	// here to avoid the possibility of leaving stale values.
	result := map[string]string{}

	if d.Destroy || d.DestroyDeposed || d.DestroyTainted {
		return result, nil
	}

	return d.applyBlockDiff(nil, attrs, schema)
}

func (d *InstanceDiff) applyBlockDiff(path []string, attrs map[string]string, schema *configschema.Block) (map[string]string, error) {
	result := map[string]string{}
	name := ""
	if len(path) > 0 {
		name = path[len(path)-1]
	}

	// localPrefix is used to build the local result map
	localPrefix := ""
	if name != "" {
		localPrefix = name + "."
	}

	// iterate over the schema rather than the attributes, so we can handle
	// different block types separately from plain attributes
	for n, attrSchema := range schema.Attributes {
		var err error
		newAttrs, err := d.applyAttrDiff(append(path, n), attrs, attrSchema)

		if err != nil {
			return result, err
		}

		for k, v := range newAttrs {
			result[localPrefix+k] = v
		}
	}

	blockPrefix := strings.Join(path, ".")
	if blockPrefix != "" {
		blockPrefix += "."
	}
	for n, block := range schema.BlockTypes {
		// we need to find the set of all keys that traverse this block
		candidateKeys := map[string]bool{}
		blockKey := blockPrefix + n + "."
		localBlockPrefix := localPrefix + n + "."

		// we can only trust the diff for sets, since the path changes, so don't
		// count existing values as candidate keys. If it turns out we're
		// keeping the attributes, we will catch it down below with "keepBlock"
		// after we check the set count.
		if block.Nesting != configschema.NestingSet {
			for k := range attrs {
				if strings.HasPrefix(k, blockKey) {
					nextDot := strings.Index(k[len(blockKey):], ".")
					if nextDot < 0 {
						continue
					}
					nextDot += len(blockKey)
					candidateKeys[k[len(blockKey):nextDot]] = true
				}
			}
		}

		for k, diff := range d.Attributes {
			if strings.HasPrefix(k, blockKey) {
				nextDot := strings.Index(k[len(blockKey):], ".")
				if nextDot < 0 {
					continue
				}

				if diff.NewRemoved {
					continue
				}

				nextDot += len(blockKey)
				candidateKeys[k[len(blockKey):nextDot]] = true
			}
		}

		// check each set candidate to see if it was removed.
		// we need to do this, because when entire sets are removed, they may
		// have the wrong key, and ony show diffs going to ""
		if block.Nesting == configschema.NestingSet {
			for k := range candidateKeys {
				indexPrefix := strings.Join(append(path, n, k), ".") + "."
				keep := false
				// now check each set element to see if it's a new diff, or one
				// that we're dropping. Since we're only applying the "New"
				// portion of the set, we can ignore diffs that only contain "Old"
				for attr, diff := range d.Attributes {
					if !strings.HasPrefix(attr, indexPrefix) {
						continue
					}

					// check for empty "count" keys
					if (strings.HasSuffix(attr, ".#") || strings.HasSuffix(attr, ".%")) && diff.New == "0" {
						continue
					}

					// removed items don't count either
					if diff.NewRemoved {
						continue
					}

					// this must be a diff to keep
					keep = true
					break
				}
				if !keep {
					delete(candidateKeys, k)
				}
			}
		}

		for k := range candidateKeys {
			newAttrs, err := d.applyBlockDiff(append(path, n, k), attrs, &block.Block)
			if err != nil {
				return result, err
			}

			for attr, v := range newAttrs {
				result[localBlockPrefix+attr] = v
			}
		}

		keepBlock := true
		// check this block's count diff directly first, since we may not
		// have candidates because it was removed and only set to "0"
		if diff, ok := d.Attributes[blockKey+"#"]; ok {
			if diff.New == "0" || diff.NewRemoved {
				keepBlock = false
			}
		}

		// if there was no diff at all, then we need to keep the block attributes
		if len(candidateKeys) == 0 && keepBlock {
			for k, v := range attrs {
				if strings.HasPrefix(k, blockKey) {
					// we need the key relative to this block, so remove the
					// entire prefix, then re-insert the block name.
					localKey := localBlockPrefix + k[len(blockKey):]
					result[localKey] = v
				}
			}
		}

		countAddr := strings.Join(append(path, n, "#"), ".")
		if countDiff, ok := d.Attributes[countAddr]; ok {
			if countDiff.NewComputed {
				result[localBlockPrefix+"#"] = hcl2shim.UnknownVariableValue
			} else {
				result[localBlockPrefix+"#"] = countDiff.New

				// While sets are complete, list are not, and we may not have all the
				// information to track removals. If the list was truncated, we need to
				// remove the extra items from the result.
				if block.Nesting == configschema.NestingList &&
					countDiff.New != "" && countDiff.New != hcl2shim.UnknownVariableValue {
					length, _ := strconv.Atoi(countDiff.New)
					for k := range result {
						if !strings.HasPrefix(k, localBlockPrefix) {
							continue
						}

						index := k[len(localBlockPrefix):]
						nextDot := strings.Index(index, ".")
						if nextDot < 1 {
							continue
						}
						index = index[:nextDot]
						i, err := strconv.Atoi(index)
						if err != nil {
							// this shouldn't happen since we added these
							// ourself, but make note of it just in case.
							log.Printf("[ERROR] bad list index in %q: %s", k, err)
							continue
						}
						if i >= length {
							delete(result, k)
						}
					}
				}
			}
		} else if origCount, ok := attrs[countAddr]; ok && keepBlock {
			result[localBlockPrefix+"#"] = origCount
		} else {
			result[localBlockPrefix+"#"] = countFlatmapContainerValues(localBlockPrefix+"#", result)
		}
	}

	return result, nil
}

func (d *InstanceDiff) applyAttrDiff(path []string, attrs map[string]string, attrSchema *configschema.Attribute) (map[string]string, error) {
	ty := attrSchema.Type
	switch {
	case ty.IsListType(), ty.IsTupleType(), ty.IsMapType():
		return d.applyCollectionDiff(path, attrs, attrSchema)
	case ty.IsSetType():
		return d.applySetDiff(path, attrs, attrSchema)
	default:
		return d.applySingleAttrDiff(path, attrs, attrSchema)
	}
}

func (d *InstanceDiff) applySingleAttrDiff(path []string, attrs map[string]string, attrSchema *configschema.Attribute) (map[string]string, error) {
	currentKey := strings.Join(path, ".")

	attr := path[len(path)-1]

	result := map[string]string{}
	diff := d.Attributes[currentKey]
	old, exists := attrs[currentKey]

	if diff != nil && diff.NewComputed {
		result[attr] = config.UnknownVariableValue
		return result, nil
	}

	// "id" must exist and not be an empty string, or it must be unknown.
	// This only applied to top-level "id" fields.
	if attr == "id" && len(path) == 1 {
		if old == "" {
			result[attr] = config.UnknownVariableValue
		} else {
			result[attr] = old
		}
		return result, nil
	}

	// attribute diffs are sometimes missed, so assume no diff means keep the
	// old value
	if diff == nil {
		if exists {
			result[attr] = old
		} else {
			// We need required values, so set those with an empty value. It
			// must be set in the config, since if it were missing it would have
			// failed validation.
			if attrSchema.Required {
				// we only set a missing string here, since bool or number types
				// would have distinct zero value which shouldn't have been
				// lost.
				if attrSchema.Type == cty.String {
					result[attr] = ""
				}
			}
		}
		return result, nil
	}

	// check for missmatched diff values
	if exists &&
		old != diff.Old &&
		old != config.UnknownVariableValue &&
		diff.Old != config.UnknownVariableValue {
		return result, fmt.Errorf("diff apply conflict for %s: diff expects %q, but prior value has %q", attr, diff.Old, old)
	}

	if diff.NewRemoved {
		// don't set anything in the new value
		return map[string]string{}, nil
	}

	if diff.Old == diff.New && diff.New == "" {
		// this can only be a valid empty string
		if attrSchema.Type == cty.String {
			result[attr] = ""
		}
		return result, nil
	}

	if attrSchema.Computed && diff.NewComputed {
		result[attr] = config.UnknownVariableValue
		return result, nil
	}

	result[attr] = diff.New

	return result, nil
}

func (d *InstanceDiff) applyCollectionDiff(path []string, attrs map[string]string, attrSchema *configschema.Attribute) (map[string]string, error) {
	result := map[string]string{}

	prefix := ""
	if len(path) > 1 {
		prefix = strings.Join(path[:len(path)-1], ".") + "."
	}

	name := ""
	if len(path) > 0 {
		name = path[len(path)-1]
	}

	currentKey := prefix + name

	// check the index first for special handling
	for k, diff := range d.Attributes {
		// check the index value, which can be set, and 0
		if k == currentKey+".#" || k == currentKey+".%" || k == currentKey {
			if diff.NewRemoved {
				return result, nil
			}

			if diff.NewComputed {
				result[k[len(prefix):]] = config.UnknownVariableValue
				return result, nil
			}

			// do what the diff tells us to here, so that it's consistent with applies
			if diff.New == "0" {
				result[k[len(prefix):]] = "0"
				return result, nil
			}
		}
	}

	// collect all the keys from the diff and the old state
	noDiff := true
	keys := map[string]bool{}
	for k := range d.Attributes {
		if !strings.HasPrefix(k, currentKey+".") {
			continue
		}
		noDiff = false
		keys[k] = true
	}

	noAttrs := true
	for k := range attrs {
		if !strings.HasPrefix(k, currentKey+".") {
			continue
		}
		noAttrs = false
		keys[k] = true
	}

	// If there's no diff and no attrs, then there's no value at all.
	// This prevents an unexpected zero-count attribute in the attributes.
	if noDiff && noAttrs {
		return result, nil
	}

	idx := "#"
	if attrSchema.Type.IsMapType() {
		idx = "%"
	}

	for k := range keys {
		// generate an schema placeholder for the values
		elSchema := &configschema.Attribute{
			Type: attrSchema.Type.ElementType(),
		}

		res, err := d.applySingleAttrDiff(append(path, k[len(currentKey)+1:]), attrs, elSchema)
		if err != nil {
			return result, err
		}

		for k, v := range res {
			result[name+"."+k] = v
		}
	}

	// Just like in nested list blocks, for simple lists we may need to fill in
	// missing empty strings.
	countKey := name + "." + idx
	count := result[countKey]
	length, _ := strconv.Atoi(count)

	if count != "" && count != hcl2shim.UnknownVariableValue &&
		attrSchema.Type.Equals(cty.List(cty.String)) {
		// insert empty strings into missing indexes
		for i := 0; i < length; i++ {
			key := fmt.Sprintf("%s.%d", name, i)
			if _, ok := result[key]; !ok {
				result[key] = ""
			}
		}
	}

	// now check for truncation in any type of list
	if attrSchema.Type.IsListType() {
		for key := range result {
			if key == countKey {
				continue
			}

			if len(key) <= len(name)+1 {
				// not sure what this is, but don't panic
				continue
			}

			index := key[len(name)+1:]

			// It is possible to have nested sets or maps, so look for another dot
			dot := strings.Index(index, ".")
			if dot > 0 {
				index = index[:dot]
			}

			// This shouldn't have any more dots, since the element type is only string.
			num, err := strconv.Atoi(index)
			if err != nil {
				log.Printf("[ERROR] bad list index in %q: %s", currentKey, err)
				continue
			}

			if num >= length {
				delete(result, key)
			}
		}
	}

	// Fill in the count value if it wasn't present in the diff for some reason,
	// or if there is no count at all.
	_, countDiff := d.Attributes[countKey]
	if result[countKey] == "" || (!countDiff && len(keys) != len(result)) {
		result[countKey] = countFlatmapContainerValues(countKey, result)
	}

	return result, nil
}

func (d *InstanceDiff) applySetDiff(path []string, attrs map[string]string, attrSchema *configschema.Attribute) (map[string]string, error) {
	// We only need this special behavior for sets of object.
	if !attrSchema.Type.ElementType().IsObjectType() {
		// The normal collection apply behavior will work okay for this one, then.
		return d.applyCollectionDiff(path, attrs, attrSchema)
	}

	// When we're dealing with a set of an object type we actually want to
	// use our normal _block type_ apply behaviors, so we'll construct ourselves
	// a synthetic schema that treats the object type as a block type and
	// then delegate to our block apply method.
	synthSchema := &configschema.Block{
		Attributes: make(map[string]*configschema.Attribute),
	}

	for name, ty := range attrSchema.Type.ElementType().AttributeTypes() {
		// We can safely make everything into an attribute here because in the
		// event that there are nested set attributes we'll end up back in
		// here again recursively and can then deal with the next level of
		// expansion.
		synthSchema.Attributes[name] = &configschema.Attribute{
			Type:     ty,
			Optional: true,
		}
	}

	parentPath := path[:len(path)-1]
	childName := path[len(path)-1]
	containerSchema := &configschema.Block{
		BlockTypes: map[string]*configschema.NestedBlock{
			childName: {
				Nesting: configschema.NestingSet,
				Block:   *synthSchema,
			},
		},
	}

	return d.applyBlockDiff(parentPath, attrs, containerSchema)
}

// countFlatmapContainerValues returns the number of values in the flatmapped container
// (set, map, list) indexed by key. The key argument is expected to include the
// trailing ".#", or ".%".
func countFlatmapContainerValues(key string, attrs map[string]string) string {
	if len(key) < 3 || !(strings.HasSuffix(key, ".#") || strings.HasSuffix(key, ".%")) {
		panic(fmt.Sprintf("invalid index value %q", key))
	}

	prefix := key[:len(key)-1]
	items := map[string]int{}

	for k := range attrs {
		if k == key {
			continue
		}
		if !strings.HasPrefix(k, prefix) {
			continue
		}

		suffix := k[len(prefix):]
		dot := strings.Index(suffix, ".")
		if dot > 0 {
			suffix = suffix[:dot]
		}

		items[suffix]++
	}
	return strconv.Itoa(len(items))
}

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
// methods but everywhere else within the terraform package.
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

		// This section builds a list of ignorable attributes for requiresNew
		// by removing off any elements of collections going to zero elements.
		// For collections going to zero, they may not exist at all in the
		// new diff (and hence RequiresNew == false).
		ignoreAttrs := make(map[string]struct{})
		for k, diffOld := range d.Attributes {
			if !strings.HasSuffix(k, ".%") && !strings.HasSuffix(k, ".#") {
				continue
			}

			// This case is in here as a protection measure. The bug that this
			// code originally fixed (GH-11349) didn't have to deal with computed
			// so I'm not 100% sure what the correct behavior is. Best to leave
			// the old behavior.
			if diffOld.NewComputed {
				continue
			}

			// We're looking for the case a map goes to exactly 0.
			if diffOld.New != "0" {
				continue
			}

			// Found it! Ignore all of these. The prefix here is stripping
			// off the "%" so it is just "k."
			prefix := k[:len(k)-1]
			for k2, _ := range d.Attributes {
				if strings.HasPrefix(k2, prefix) {
					ignoreAttrs[k2] = struct{}{}
				}
			}
		}

		for k, rd := range d.Attributes {
			if _, ok := ignoreAttrs[k]; ok {
				continue
			}

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

		// We don't compare the values because we can't currently actually
		// guarantee to generate the same value two two diffs created from
		// the same state+config: we have some pesky interpolation functions
		// that do not behave as pure functions (uuid, timestamp) and so they
		// can be different each time a diff is produced.
		// FIXME: Re-organize our config handling so that we don't re-evaluate
		// expressions when we produce a second comparison diff during
		// apply (for EvalCompareDiff).
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
