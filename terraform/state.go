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
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/config"
)

const (
	// StateVersion is the current version for our state file
	StateVersion = 1
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

	// Remote is used to track the metadata required to
	// pull and push state files from a remote storage endpoint.
	Remote *RemoteState `json:"remote,omitempty"`

	// Modules contains all the modules in a breadth-first order
	Modules []*ModuleState `json:"modules"`
}

// NewState is used to initialize a blank state
func NewState() *State {
	s := &State{}
	s.init()
	return s
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
// This should be the preferred lookup mechanism as it allows for future
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

// ModuleOrphans returns all the module orphans in this state by
// returning their full paths. These paths can be used with ModuleByPath
// to return the actual state.
func (s *State) ModuleOrphans(path []string, c *config.Config) [][]string {
	// direct keeps track of what direct children we have both in our config
	// and in our state. childrenKeys keeps track of what isn't an orphan.
	direct := make(map[string]struct{})
	childrenKeys := make(map[string]struct{})
	if c != nil {
		for _, m := range c.Modules {
			childrenKeys[m.Name] = struct{}{}
			direct[m.Name] = struct{}{}
		}
	}

	// Go over the direct children and find any that aren't in our keys.
	var orphans [][]string
	for _, m := range s.Children(path) {
		key := m.Path[len(m.Path)-1]

		// Record that we found this key as a direct child. We use this
		// later to find orphan nested modules.
		direct[key] = struct{}{}

		// If we have a direct child still in our config, it is not an orphan
		if _, ok := childrenKeys[key]; ok {
			continue
		}

		orphans = append(orphans, m.Path)
	}

	// Find the orphans that are nested...
	for _, m := range s.Modules {
		// We only want modules that are at least grandchildren
		if len(m.Path) < len(path)+2 {
			continue
		}

		// If it isn't part of our tree, continue
		if !reflect.DeepEqual(path, m.Path[:len(path)]) {
			continue
		}

		// If we have the direct child, then just skip it.
		key := m.Path[len(path)]
		if _, ok := direct[key]; ok {
			continue
		}

		orphanPath := m.Path[:len(path)+1]

		// Don't double-add if we've already added this orphan (which can happen if
		// there are multiple nested sub-modules that get orphaned together).
		alreadyAdded := false
		for _, o := range orphans {
			if reflect.DeepEqual(o, orphanPath) {
				alreadyAdded = true
				break
			}
		}
		if alreadyAdded {
			continue
		}

		// Add this orphan
		orphans = append(orphans, orphanPath)
	}

	return orphans
}

// Empty returns true if the state is empty.
func (s *State) Empty() bool {
	if s == nil {
		return true
	}

	return len(s.Modules) == 0
}

// IsRemote returns true if State represents a state that exists and is
// remote.
func (s *State) IsRemote() bool {
	if s == nil {
		return false
	}
	if s.Remote == nil {
		return false
	}
	if s.Remote.Type == "" {
		return false
	}

	return true
}

// RootModule returns the ModuleState for the root module
func (s *State) RootModule() *ModuleState {
	root := s.ModuleByPath(rootModulePath)
	if root == nil {
		panic("missing root module")
	}
	return root
}

// Equal tests if one state is equal to another.
func (s *State) Equal(other *State) bool {
	// If one is nil, we do a direct check
	if s == nil || other == nil {
		return s == other
	}

	// If the versions are different, they're certainly not equal
	if s.Version != other.Version {
		return false
	}

	// If any of the modules are not equal, then this state isn't equal
	if len(s.Modules) != len(other.Modules) {
		return false
	}
	for _, m := range s.Modules {
		// This isn't very optimal currently but works.
		otherM := other.ModuleByPath(m.Path)
		if otherM == nil {
			return false
		}

		// If they're not equal, then we're not equal!
		if !m.Equal(otherM) {
			return false
		}
	}

	return true
}

// DeepCopy performs a deep copy of the state structure and returns
// a new structure.
func (s *State) DeepCopy() *State {
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
	if s.Remote != nil {
		n.Remote = s.Remote.deepcopy()
	}
	return n
}

// IncrementSerialMaybe increments the serial number of this state
// if it different from the other state.
func (s *State) IncrementSerialMaybe(other *State) {
	if s == nil {
		return
	}
	if other == nil {
		return
	}
	if s.Serial > other.Serial {
		return
	}
	if !s.Equal(other) {
		if other.Serial > s.Serial {
			s.Serial = other.Serial
		}

		s.Serial++
	}
}

func (s *State) init() {
	if s.Version == 0 {
		s.Version = StateVersion
	}
	if len(s.Modules) == 0 {
		root := &ModuleState{
			Path: rootModulePath,
		}
		root.init()
		s.Modules = []*ModuleState{root}
	}
}

// prune is used to remove any resources that are no longer required
func (s *State) prune() {
	if s == nil {
		return
	}
	for _, mod := range s.Modules {
		mod.prune()
	}
	if s.Remote != nil && s.Remote.Empty() {
		s.Remote = nil
	}
}

// sort sorts the modules
func (s *State) sort() {
	sort.Sort(moduleStateSort(s.Modules))

	// Allow modules to be sorted
	for _, m := range s.Modules {
		m.sort()
	}
}

func (s *State) GoString() string {
	return fmt.Sprintf("*%#v", *s)
}

func (s *State) String() string {
	if s == nil {
		return "<nil>"
	}

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
			text := s.Text()
			if text != "" {
				text = "  " + text
			}

			buf.WriteString(fmt.Sprintf("%s\n", text))
		}
	}

	return strings.TrimSpace(buf.String())
}

// RemoteState is used to track the information about a remote
// state store that we push/pull state to.
type RemoteState struct {
	// Type controls the client we use for the remote state
	Type string `json:"type"`

	// Config is used to store arbitrary configuration that
	// is type specific
	Config map[string]string `json:"config"`
}

func (r *RemoteState) deepcopy() *RemoteState {
	confCopy := make(map[string]string, len(r.Config))
	for k, v := range r.Config {
		confCopy[k] = v
	}
	return &RemoteState{
		Type:   r.Type,
		Config: confCopy,
	}
}

func (r *RemoteState) Empty() bool {
	return r == nil || r.Type == ""
}

func (r *RemoteState) Equals(other *RemoteState) bool {
	if r.Type != other.Type {
		return false
	}
	if len(r.Config) != len(other.Config) {
		return false
	}
	for k, v := range r.Config {
		if other.Config[k] != v {
			return false
		}
	}
	return true
}

func (r *RemoteState) GoString() string {
	return fmt.Sprintf("*%#v", *r)
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

// Equal tests whether one module state is equal to another.
func (m *ModuleState) Equal(other *ModuleState) bool {
	// Paths must be equal
	if !reflect.DeepEqual(m.Path, other.Path) {
		return false
	}

	// Outputs must be equal
	if len(m.Outputs) != len(other.Outputs) {
		return false
	}
	for k, v := range m.Outputs {
		if other.Outputs[k] != v {
			return false
		}
	}

	// Dependencies must be equal. This sorts these in place but
	// this shouldn't cause any problems.
	sort.Strings(m.Dependencies)
	sort.Strings(other.Dependencies)
	if len(m.Dependencies) != len(other.Dependencies) {
		return false
	}
	for i, d := range m.Dependencies {
		if other.Dependencies[i] != d {
			return false
		}
	}

	// Resources must be equal
	if len(m.Resources) != len(other.Resources) {
		return false
	}
	for k, r := range m.Resources {
		otherR, ok := other.Resources[k]
		if !ok {
			return false
		}

		if !r.Equal(otherR) {
			return false
		}
	}

	return true
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

	if c != nil {
		for _, r := range c.Resources {
			delete(keys, r.Id())

			for k, _ := range keys {
				if strings.HasPrefix(k, r.Id()+".") {
					delete(keys, k)
				}
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

		if (v.Primary == nil || v.Primary.ID == "") && len(v.Tainted) == 0 && len(v.Deposed) == 0 {
			delete(m.Resources, k)
		}
	}

	for k, v := range m.Outputs {
		if v == config.UnknownVariableValue {
			delete(m.Outputs, k)
		}
	}
}

func (m *ModuleState) sort() {
	for _, v := range m.Resources {
		v.sort()
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

		deposedStr := ""
		if len(rs.Deposed) > 0 {
			deposedStr = fmt.Sprintf(" (%d deposed)", len(rs.Deposed))
		}

		buf.WriteString(fmt.Sprintf("%s:%s%s\n", k, taintStr, deposedStr))
		buf.WriteString(fmt.Sprintf("  ID = %s\n", id))
		if rs.Provider != "" {
			buf.WriteString(fmt.Sprintf("  provider = %s\n", rs.Provider))
		}

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

		for idx, t := range rs.Deposed {
			buf.WriteString(fmt.Sprintf("  Deposed ID %d = %s\n", idx+1, t.ID))
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

// ResourceStateKey is a structured representation of the key used for the
// ModuleState.Resources mapping
type ResourceStateKey struct {
	Name  string
	Type  string
	Index int
}

// Equal determines whether two ResourceStateKeys are the same
func (rsk *ResourceStateKey) Equal(other *ResourceStateKey) bool {
	if rsk == nil || other == nil {
		return false
	}
	if rsk.Type != other.Type {
		return false
	}
	if rsk.Name != other.Name {
		return false
	}
	if rsk.Index != other.Index {
		return false
	}
	return true
}

func (rsk *ResourceStateKey) String() string {
	if rsk == nil {
		return ""
	}
	if rsk.Index == -1 {
		return fmt.Sprintf("%s.%s", rsk.Type, rsk.Name)
	}
	return fmt.Sprintf("%s.%s.%d", rsk.Type, rsk.Name, rsk.Index)
}

// ParseResourceStateKey accepts a key in the format used by
// ModuleState.Resources and returns a resource name and resource index. In the
// state, a resource has the format "type.name.index" or "type.name". In the
// latter case, the index is returned as -1.
func ParseResourceStateKey(k string) (*ResourceStateKey, error) {
	parts := strings.Split(k, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return nil, fmt.Errorf("Malformed resource state key: %s", k)
	}
	rsk := &ResourceStateKey{
		Type:  parts[0],
		Name:  parts[1],
		Index: -1,
	}
	if len(parts) == 3 {
		index, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("Malformed resource state key index: %s", k)
		}
		rsk.Index = index
	}
	return rsk, nil
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

	// Deposed is used in the mechanics of CreateBeforeDestroy: the existing
	// Primary is Deposed to get it out of the way for the replacement Primary to
	// be created by Apply. If the replacement Primary creates successfully, the
	// Deposed instance is cleaned up. If there were problems creating the
	// replacement, the instance remains in the Deposed list so it can be
	// destroyed in a future run. Functionally, Deposed instances are very
	// similar to Tainted instances in that Terraform is only tracking them in
	// order to remember to destroy them.
	Deposed []*InstanceState `json:"deposed,omitempty"`

	// Provider is used when a resource is connected to a provider with an alias.
	// If this string is empty, the resource is connected to the default provider,
	// e.g. "aws_instance" goes with the "aws" provider.
	// If the resource block contained a "provider" key, that value will be set here.
	Provider string `json:"provider,omitempty"`
}

// Equal tests whether two ResourceStates are equal.
func (s *ResourceState) Equal(other *ResourceState) bool {
	if s.Type != other.Type {
		return false
	}

	if s.Provider != other.Provider {
		return false
	}

	// Dependencies must be equal
	sort.Strings(s.Dependencies)
	sort.Strings(other.Dependencies)
	if len(s.Dependencies) != len(other.Dependencies) {
		return false
	}
	for i, d := range s.Dependencies {
		if other.Dependencies[i] != d {
			return false
		}
	}

	// States must be equal
	if !s.Primary.Equal(other.Primary) {
		return false
	}

	// Tainted
	taints := make(map[string]*InstanceState)
	for _, t := range other.Tainted {
		if t == nil {
			continue
		}

		taints[t.ID] = t
	}
	for _, t := range s.Tainted {
		if t == nil {
			continue
		}

		otherT, ok := taints[t.ID]
		if !ok {
			return false
		}
		delete(taints, t.ID)

		if !t.Equal(otherT) {
			return false
		}
	}

	// This means that we have stuff in other tainted that we don't
	// have, so it is not equal.
	if len(taints) > 0 {
		return false
	}

	return true
}

// Taint takes the primary state and marks it as tainted. If there is no
// primary state, this does nothing.
func (r *ResourceState) Taint() {
	// If there is no primary, nothing to do
	if r.Primary == nil {
		return
	}

	// Shuffle to the end of the taint list and set primary to nil
	r.Tainted = append(r.Tainted, r.Primary)
	r.Primary = nil
}

// Untaint takes a tainted InstanceState and marks it as primary.
// The index argument is used to select a single InstanceState from the
// array of Tainted when there are more than one. If index is -1, the
// first Tainted InstanceState will be untainted iff there is only one
// Tainted InstanceState. Index must be >= 0 to specify an InstanceState
// when Tainted has more than one member.
func (r *ResourceState) Untaint(index int) error {
	if len(r.Tainted) == 0 {
		return fmt.Errorf("Nothing to untaint.")
	}
	if r.Primary != nil {
		return fmt.Errorf("Resource has a primary instance in the state that would be overwritten by untainting. If you want to restore a tainted resource to primary, taint the existing primary instance first.")
	}
	if index == -1 && len(r.Tainted) > 1 {
		return fmt.Errorf("There are %d tainted instances for this resource, please specify an index to select which one to untaint.", len(r.Tainted))
	}
	if index == -1 {
		index = 0
	}
	if index >= len(r.Tainted) {
		return fmt.Errorf("There are %d tainted instances for this resource, the index specified (%d) is out of range.", len(r.Tainted), index)
	}

	// Perform the untaint
	r.Primary = r.Tainted[index]
	r.Tainted = append(r.Tainted[:index], r.Tainted[index+1:]...)

	return nil
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
		Dependencies: nil,
		Primary:      r.Primary.deepcopy(),
		Tainted:      nil,
		Provider:     r.Provider,
	}
	if r.Dependencies != nil {
		n.Dependencies = make([]string, len(r.Dependencies))
		copy(n.Dependencies, r.Dependencies)
	}
	if r.Tainted != nil {
		n.Tainted = make([]*InstanceState, 0, len(r.Tainted))
		for _, inst := range r.Tainted {
			n.Tainted = append(n.Tainted, inst.deepcopy())
		}
	}
	if r.Deposed != nil {
		n.Deposed = make([]*InstanceState, 0, len(r.Deposed))
		for _, inst := range r.Deposed {
			n.Deposed = append(n.Deposed, inst.deepcopy())
		}
	}

	return n
}

// prune is used to remove any instances that are no longer required
func (r *ResourceState) prune() {
	n := len(r.Tainted)
	for i := 0; i < n; i++ {
		inst := r.Tainted[i]
		if inst == nil || inst.ID == "" {
			copy(r.Tainted[i:], r.Tainted[i+1:])
			r.Tainted[n-1] = nil
			n--
			i--
		}
	}

	r.Tainted = r.Tainted[:n]

	n = len(r.Deposed)
	for i := 0; i < n; i++ {
		inst := r.Deposed[i]
		if inst == nil || inst.ID == "" {
			copy(r.Deposed[i:], r.Deposed[i+1:])
			r.Deposed[n-1] = nil
			n--
			i--
		}
	}

	r.Deposed = r.Deposed[:n]
}

func (r *ResourceState) sort() {
	sort.Strings(r.Dependencies)
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

	// Meta is a simple K/V map that is persisted to the State but otherwise
	// ignored by Terraform core. It's meant to be used for accounting by
	// external client code.
	Meta map[string]string `json:"meta,omitempty"`
}

func (i *InstanceState) init() {
	if i.Attributes == nil {
		i.Attributes = make(map[string]string)
	}
	if i.Meta == nil {
		i.Meta = make(map[string]string)
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
	if i.Meta != nil {
		n.Meta = make(map[string]string, len(i.Meta))
		for k, v := range i.Meta {
			n.Meta[k] = v
		}
	}
	return n
}

func (s *InstanceState) Empty() bool {
	return s == nil || s.ID == ""
}

func (s *InstanceState) Equal(other *InstanceState) bool {
	// Short circuit some nil checks
	if s == nil || other == nil {
		return s == other
	}

	// IDs must be equal
	if s.ID != other.ID {
		return false
	}

	// Attributes must be equal
	if len(s.Attributes) != len(other.Attributes) {
		return false
	}
	for k, v := range s.Attributes {
		otherV, ok := other.Attributes[k]
		if !ok {
			return false
		}

		if v != otherV {
			return false
		}
	}

	// Meta must be equal
	if len(s.Meta) != len(other.Meta) {
		return false
	}
	for k, v := range s.Meta {
		otherV, ok := other.Meta[k]
		if !ok {
			return false
		}

		if v != otherV {
			return false
		}
	}

	return true
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
	if state.Version > StateVersion {
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
	d.Version = StateVersion

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

	// Otherwise, compare lexically
	return strings.Join(a.Path, ".") < strings.Join(b.Path, ".")
}

func (s moduleStateSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
