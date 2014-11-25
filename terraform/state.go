package terraform

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/config"
)

const (
	// textStateVersion is the current version for our state file
	textStateVersion = 1
)

// rootModulePath is the path of the root module
var rootModulePath = []string{"root"}

// State keeps track of a snapshot state-of-the-world that Terraform
// can use to keep track of what real world resources it is actually
// managing. This is the latest format as of Terraform 0.3
type State struct {
	// Version is the protocol version. Currently only "1".
	Version int `json:"version"`

	// Serial is incremented on any operation that modifies
	// the State file. It is used to detect potentially conflicting
	// updates.
	Serial int64 `json:"serial"`

	// Modules contains all the modules in a breadth-first order
	Modules []*ModuleState `json:"modules"`
}

// Children returns the ModuleStates that are direct children of
// the given path. If the path is "root", for example, then children
// returned might be "root.child", but not "root.child.grandchild".
func (s *State) Children(path []string) []*ModuleState {
	// TODO: test

	result := make([]*ModuleState, 0)
	for _, m := range s.Modules {
		if len(m.Path) != len(path)+1 {
			continue
		}
		if !reflect.DeepEqual(path, m.Path[:len(path)]) {
			continue
		}

		result = append(result, m)
	}

	return result
}

// AddModule adds the module with the given path to the state.
//
// This should be the preferred method to add module states since it
// allows us to optimize lookups later as well as control sorting.
func (s *State) AddModule(path []string) *ModuleState {
	m := &ModuleState{Path: path}
	m.init()
	s.Modules = append(s.Modules, m)
	s.sort()
	return m
}

// ModuleByPath is used to lookup the module state for the given path.
// This should be the prefered lookup mechanism as it allows for future
// lookup optimizations.
func (s *State) ModuleByPath(path []string) *ModuleState {
	if s == nil {
		return nil
	}
	for _, mod := range s.Modules {
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
func (s *State) RootModule() *ModuleState {
	root := s.ModuleByPath(rootModulePath)
	if root == nil {
		panic("missing root module")
	}
	return root
}

func (s *State) init() {
	if s.Version == 0 {
		s.Version = textStateVersion
	}
	if len(s.Modules) == 0 {
		root := &ModuleState{
			Path: rootModulePath,
		}
		root.init()
		s.Modules = []*ModuleState{root}
	}
}

func (s *State) deepcopy() *State {
	if s == nil {
		return nil
	}
	n := &State{
		Version: s.Version,
		Serial:  s.Serial,
		Modules: make([]*ModuleState, 0, len(s.Modules)),
	}
	for _, mod := range s.Modules {
		n.Modules = append(n.Modules, mod.deepcopy())
	}
	return n
}

// prune is used to remove any resources that are no longer required
func (s *State) prune() {
	if s == nil {
		return
	}
	for _, mod := range s.Modules {
		mod.prune()
	}
}

// sort sorts the modules
func (s *State) sort() {
	sort.Sort(moduleStateSort(s.Modules))
}

func (s *State) GoString() string {
	return fmt.Sprintf("*%#v", *s)
}

func (s *State) String() string {
	var buf bytes.Buffer
	for _, m := range s.Modules {
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

// ModuleState is used to track all the state relevant to a single
// module. Previous to Terraform 0.3, all state belonged to the "root"
// module.
type ModuleState struct {
	// Path is the import path from the root module. Modules imports are
	// always disjoint, so the path represents amodule tree
	Path []string `json:"path"`

	// Outputs declared by the module and maintained for each module
	// even though only the root module technically needs to be kept.
	// This allows operators to inspect values at the boundaries.
	Outputs map[string]string `json:"outputs"`

	// Resources is a mapping of the logically named resource to
	// the state of the resource. Each resource may actually have
	// N instances underneath, although a user only needs to think
	// about the 1:1 case.
	Resources map[string]*ResourceState `json:"resources"`

	// Dependencies are a list of things that this module relies on
	// existing to remain intact. For example: an module may depend
	// on a VPC ID given by an aws_vpc resource.
	//
	// Terraform uses this information to build valid destruction
	// orders and to warn the user if they're destroying a module that
	// another resource depends on.
	//
	// Things can be put into this list that may not be managed by
	// Terraform. If Terraform doesn't find a matching ID in the
	// overall state, then it assumes it isn't managed and doesn't
	// worry about it.
	Dependencies []string `json:"depends_on,omitempty"`
}

// IsRoot says whether or not this module diff is for the root module.
func (m *ModuleState) IsRoot() bool {
	return reflect.DeepEqual(m.Path, rootModulePath)
}

// Orphans returns a list of keys of resources that are in the State
// but aren't present in the configuration itself. Hence, these keys
// represent the state of resources that are orphans.
func (m *ModuleState) Orphans(c *config.Config) []string {
	keys := make(map[string]struct{})
	for k, _ := range m.Resources {
		keys[k] = struct{}{}
	}

	for _, r := range c.Resources {
		delete(keys, r.Id())

		for k, _ := range keys {
			if strings.HasPrefix(k, r.Id()+".") {
				delete(keys, k)
			}
		}
	}

	result := make([]string, 0, len(keys))
	for k, _ := range keys {
		result = append(result, k)
	}

	return result
}

// View returns a view with the given resource prefix.
func (m *ModuleState) View(id string) *ModuleState {
	if m == nil {
		return m
	}

	r := m.deepcopy()
	for k, _ := range r.Resources {
		if id == k || strings.HasPrefix(k, id+".") {
			continue
		}

		delete(r.Resources, k)
	}

	return r
}

func (m *ModuleState) init() {
	if m.Outputs == nil {
		m.Outputs = make(map[string]string)
	}
	if m.Resources == nil {
		m.Resources = make(map[string]*ResourceState)
	}
}

func (m *ModuleState) deepcopy() *ModuleState {
	if m == nil {
		return nil
	}
	n := &ModuleState{
		Path:      make([]string, len(m.Path)),
		Outputs:   make(map[string]string, len(m.Outputs)),
		Resources: make(map[string]*ResourceState, len(m.Resources)),
	}
	copy(n.Path, m.Path)
	for k, v := range m.Outputs {
		n.Outputs[k] = v
	}
	for k, v := range m.Resources {
		n.Resources[k] = v.deepcopy()
	}
	return n
}

// prune is used to remove any resources that are no longer required
func (m *ModuleState) prune() {
	for k, v := range m.Resources {
		v.prune()
		if (v.Primary == nil || v.Primary.ID == "") && len(v.Tainted) == 0 {
			delete(m.Resources, k)
		}
	}
}

func (m *ModuleState) GoString() string {
	return fmt.Sprintf("*%#v", *m)
}

func (m *ModuleState) String() string {
	var buf bytes.Buffer

	if len(m.Resources) == 0 {
		buf.WriteString("<no state>")
	}

	names := make([]string, 0, len(m.Resources))
	for name, _ := range m.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, k := range names {
		rs := m.Resources[k]
		var id string
		if rs.Primary != nil {
			id = rs.Primary.ID
		}
		if id == "" {
			id = "<not created>"
		}

		taintStr := ""
		if len(rs.Tainted) > 0 {
			taintStr = fmt.Sprintf(" (%d tainted)", len(rs.Tainted))
		}

		buf.WriteString(fmt.Sprintf("%s:%s\n", k, taintStr))
		buf.WriteString(fmt.Sprintf("  ID = %s\n", id))

		var attributes map[string]string
		if rs.Primary != nil {
			attributes = rs.Primary.Attributes
		}
		attrKeys := make([]string, 0, len(attributes))
		for ak, _ := range attributes {
			if ak == "id" {
				continue
			}

			attrKeys = append(attrKeys, ak)
		}
		sort.Strings(attrKeys)

		for _, ak := range attrKeys {
			av := attributes[ak]
			buf.WriteString(fmt.Sprintf("  %s = %s\n", ak, av))
		}

		for idx, t := range rs.Tainted {
			buf.WriteString(fmt.Sprintf("  Tainted ID %d = %s\n", idx+1, t.ID))
		}

		if len(rs.Dependencies) > 0 {
			buf.WriteString(fmt.Sprintf("\n  Dependencies:\n"))
			for _, dep := range rs.Dependencies {
				buf.WriteString(fmt.Sprintf("    %s\n", dep))
			}
		}
	}

	if len(m.Outputs) > 0 {
		buf.WriteString("\nOutputs:\n\n")

		ks := make([]string, 0, len(m.Outputs))
		for k, _ := range m.Outputs {
			ks = append(ks, k)
		}
		sort.Strings(ks)

		for _, k := range ks {
			v := m.Outputs[k]
			buf.WriteString(fmt.Sprintf("%s = %s\n", k, v))
		}
	}

	return buf.String()
}

// ResourceState holds the state of a resource that is used so that
// a provider can find and manage an existing resource as well as for
// storing attributes that are used to populate variables of child
// resources.
//
// Attributes has attributes about the created resource that are
// queryable in interpolation: "${type.id.attr}"
//
// Extra is just extra data that a provider can return that we store
// for later, but is not exposed in any way to the user.
//
type ResourceState struct {
	// This is filled in and managed by Terraform, and is the resource
	// type itself such as "mycloud_instance". If a resource provider sets
	// this value, it won't be persisted.
	Type string `json:"type"`

	// Dependencies are a list of things that this resource relies on
	// existing to remain intact. For example: an AWS instance might
	// depend on a subnet (which itself might depend on a VPC, and so
	// on).
	//
	// Terraform uses this information to build valid destruction
	// orders and to warn the user if they're destroying a resource that
	// another resource depends on.
	//
	// Things can be put into this list that may not be managed by
	// Terraform. If Terraform doesn't find a matching ID in the
	// overall state, then it assumes it isn't managed and doesn't
	// worry about it.
	Dependencies []string `json:"depends_on,omitempty"`

	// Primary is the current active instance for this resource.
	// It can be replaced but only after a successful creation.
	// This is the instances on which providers will act.
	Primary *InstanceState `json:"primary"`

	// Tainted is used to track any underlying instances that
	// have been created but are in a bad or unknown state and
	// need to be cleaned up subsequently.  In the
	// standard case, there is only at most a single instance.
	// However, in pathological cases, it is possible for the number
	// of instances to accumulate.
	Tainted []*InstanceState `json:"tainted,omitempty"`
}

func (r *ResourceState) init() {
	if r.Primary == nil {
		r.Primary = &InstanceState{}
	}
	r.Primary.init()
}

func (r *ResourceState) deepcopy() *ResourceState {
	if r == nil {
		return nil
	}
	n := &ResourceState{
		Type:         r.Type,
		Dependencies: make([]string, len(r.Dependencies)),
		Primary:      r.Primary.deepcopy(),
		Tainted:      make([]*InstanceState, 0, len(r.Tainted)),
	}
	copy(n.Dependencies, r.Dependencies)
	for _, inst := range r.Tainted {
		n.Tainted = append(n.Tainted, inst.deepcopy())
	}
	return n
}

// prune is used to remove any instances that are no longer required
func (r *ResourceState) prune() {
	n := len(r.Tainted)
	for i := 0; i < n; i++ {
		inst := r.Tainted[i]
		if inst.ID == "" {
			copy(r.Tainted[i:], r.Tainted[i+1:])
			r.Tainted[n-1] = nil
			n--
		}
	}
	r.Tainted = r.Tainted[:n]
}

func (s *ResourceState) GoString() string {
	return fmt.Sprintf("*%#v", *s)
}

func (s *ResourceState) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("Type = %s", s.Type))
	return buf.String()
}

// InstanceState is used to track the unique state information belonging
// to a given instance.
type InstanceState struct {
	// A unique ID for this resource. This is opaque to Terraform
	// and is only meant as a lookup mechanism for the providers.
	ID string `json:"id"`

	// Attributes are basic information about the resource. Any keys here
	// are accessible in variable format within Terraform configurations:
	// ${resourcetype.name.attribute}.
	Attributes map[string]string `json:"attributes,omitempty"`

	// Ephemeral is used to store any state associated with this instance
	// that is necessary for the Terraform run to complete, but is not
	// persisted to a state file.
	Ephemeral EphemeralState `json:"-"`
}

func (i *InstanceState) init() {
	if i.Attributes == nil {
		i.Attributes = make(map[string]string)
	}
	i.Ephemeral.init()
}

func (i *InstanceState) deepcopy() *InstanceState {
	if i == nil {
		return nil
	}
	n := &InstanceState{
		ID:        i.ID,
		Ephemeral: *i.Ephemeral.deepcopy(),
	}
	if i.Attributes != nil {
		n.Attributes = make(map[string]string, len(i.Attributes))
		for k, v := range i.Attributes {
			n.Attributes[k] = v
		}
	}
	return n
}

// MergeDiff takes a ResourceDiff and merges the attributes into
// this resource state in order to generate a new state. This new
// state can be used to provide updated attribute lookups for
// variable interpolation.
//
// If the diff attribute requires computing the value, and hence
// won't be available until apply, the value is replaced with the
// computeID.
func (s *InstanceState) MergeDiff(d *InstanceDiff) *InstanceState {
	result := s.deepcopy()
	if result == nil {
		result = new(InstanceState)
	}
	result.init()

	if s != nil {
		for k, v := range s.Attributes {
			result.Attributes[k] = v
		}
	}
	if d != nil {
		for k, diff := range d.Attributes {
			if diff.NewRemoved {
				delete(result.Attributes, k)
				continue
			}
			if diff.NewComputed {
				result.Attributes[k] = config.UnknownVariableValue
				continue
			}

			result.Attributes[k] = diff.New
		}
	}

	return result
}

func (i *InstanceState) GoString() string {
	return fmt.Sprintf("*%#v", *i)
}

func (i *InstanceState) String() string {
	var buf bytes.Buffer

	if i == nil || i.ID == "" {
		return "<not created>"
	}

	buf.WriteString(fmt.Sprintf("ID = %s\n", i.ID))

	attributes := i.Attributes
	attrKeys := make([]string, 0, len(attributes))
	for ak, _ := range attributes {
		if ak == "id" {
			continue
		}

		attrKeys = append(attrKeys, ak)
	}
	sort.Strings(attrKeys)

	for _, ak := range attrKeys {
		av := attributes[ak]
		buf.WriteString(fmt.Sprintf("%s = %s\n", ak, av))
	}

	return buf.String()
}

// EphemeralState is used for transient state that is only kept in-memory
type EphemeralState struct {
	// ConnInfo is used for the providers to export information which is
	// used to connect to the resource for provisioning. For example,
	// this could contain SSH or WinRM credentials.
	ConnInfo map[string]string `json:"-"`
}

func (e *EphemeralState) init() {
	if e.ConnInfo == nil {
		e.ConnInfo = make(map[string]string)
	}
}

func (e *EphemeralState) deepcopy() *EphemeralState {
	if e == nil {
		return nil
	}
	n := &EphemeralState{}
	if e.ConnInfo != nil {
		n.ConnInfo = make(map[string]string, len(e.ConnInfo))
		for k, v := range e.ConnInfo {
			n.ConnInfo[k] = v
		}
	}
	return n
}

// ReadState reads a state structure out of a reader in the format that
// was written by WriteState.
func ReadState(src io.Reader) (*State, error) {
	buf := bufio.NewReader(src)

	// Check if this is a V1 format
	start, err := buf.Peek(len(stateFormatMagic))
	if err != nil {
		return nil, fmt.Errorf("Failed to check for magic bytes: %v", err)
	}
	if string(start) == stateFormatMagic {
		// Read the old state
		old, err := ReadStateV1(buf)
		if err != nil {
			return nil, err
		}
		return upgradeV1State(old)
	}

	// Otherwise, must be V2
	dec := json.NewDecoder(buf)
	state := &State{}
	if err := dec.Decode(state); err != nil {
		return nil, fmt.Errorf("Decoding state file failed: %v", err)
	}

	// Check the version, this to ensure we don't read a future
	// version that we don't understand
	if state.Version > textStateVersion {
		return nil, fmt.Errorf("State version %d not supported, please update.",
			state.Version)
	}

	// Sort it
	state.sort()

	return state, nil
}

// WriteState writes a state somewhere in a binary format.
func WriteState(d *State, dst io.Writer) error {
	// Make sure it is sorted
	d.sort()

	// Ensure the version is set
	d.Version = textStateVersion

	// Always increment the serial number
	d.Serial++

	// Encode the data in a human-friendly way
	data, err := json.MarshalIndent(d, "", "    ")
	if err != nil {
		return fmt.Errorf("Failed to encode state: %s", err)
	}

	// We append a newline to the data because MarshalIndent doesn't
	data = append(data, '\n')

	// Write the data out to the dst
	if _, err := io.Copy(dst, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("Failed to write state: %v", err)
	}

	return nil
}

// upgradeV1State is used to upgrade a V1 state representation
// into a proper State representation.
func upgradeV1State(old *StateV1) (*State, error) {
	s := &State{}
	s.init()

	// Old format had no modules, so we migrate everything
	// directly into the root module.
	root := s.RootModule()

	// Copy the outputs
	root.Outputs = old.Outputs

	// Upgrade the resources
	for id, rs := range old.Resources {
		newRs := &ResourceState{
			Type: rs.Type,
		}
		root.Resources[id] = newRs

		// Migrate to an instance state
		instance := &InstanceState{
			ID:         rs.ID,
			Attributes: rs.Attributes,
		}

		// Check if this is the primary or tainted instance
		if _, ok := old.Tainted[id]; ok {
			newRs.Tainted = append(newRs.Tainted, instance)
		} else {
			newRs.Primary = instance
		}

		// Warn if the resource uses Extra, as there is
		// no upgrade path for this! Now totally deprecated.
		if len(rs.Extra) > 0 {
			log.Printf(
				"[WARN] Resource %s uses deprecated attribute "+
					"storage, state file upgrade may be incomplete.",
				rs.ID,
			)
		}
	}
	return s, nil
}

// moduleStateSort implements sort.Interface to sort module states
type moduleStateSort []*ModuleState

func (s moduleStateSort) Len() int {
	return len(s)
}

func (s moduleStateSort) Less(i, j int) bool {
	a := s[i]
	b := s[j]

	// If the lengths are different, then the shorter one always wins
	if len(a.Path) != len(b.Path) {
		return len(a.Path) < len(b.Path)
	}

	// Otherwise, compare by last path element
	idx := len(a.Path) - 1
	return a.Path[idx] < b.Path[idx]
}

func (s moduleStateSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
