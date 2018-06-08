package states

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/hashicorp/terraform/addrs"
)

// Resource represents the state of a resource.
type Resource struct {
	// Addr is the module-relative address for the resource this state object
	// belongs to.
	Addr addrs.Resource

	// EachMode is the multi-instance mode currently in use for this resource,
	// or NoEach if this is a single-instance resource. This dictates what
	// type of value is returned when accessing this resource via expressions
	// in the Terraform language.
	EachMode EachMode

	// Instances contains the potentially-multiple instances associated with
	// this resource. This map can contain a mixture of different key types,
	// but only the ones of InstanceKeyType are considered current.
	Instances map[addrs.InstanceKey]*ResourceInstance

	// ProviderConfig is the absolute address for the provider configuration that
	// most recently managed this resource. This is used to connect a resource
	// with a provider configuration when the resource configuration block is
	// not available, such as if it has been removed from configuration
	// altogether.
	ProviderConfig addrs.AbsProviderConfig
}

// Instance returns the state for the instance with the given key, or nil
// if no such instance is tracked within the state.
func (rs *Resource) Instance(key addrs.InstanceKey) *ResourceInstance {
	return rs.Instances[key]
}

// EnsureInstance returns the state for the instance with the given key,
// creating a new empty state for it if one doesn't already exist.
//
// Because this may create and save a new state, it is considered to be
// a write operation.
func (rs *Resource) EnsureInstance(key addrs.InstanceKey) *ResourceInstance {
	ret := rs.Instance(key)
	if ret == nil {
		ret = NewResourceInstance()
		rs.Instances[key] = ret
	}
	return ret
}

// ResourceInstance represents the state of a particular instance of a resource.
type ResourceInstance struct {
	// Current, if non-nil, is the remote object that is currently represented
	// by the corresponding resource instance.
	Current *ResourceInstanceObject

	// Deposed, if len > 0, contains any remote objects that were previously
	// represented by the corresponding resource instance but have been
	// replaced and are pending destruction due to the create_before_destroy
	// lifecycle mode.
	Deposed map[DeposedKey]*ResourceInstanceObject
}

// NewResourceInstance constructs and returns a new ResourceInstance, ready to
// use.
func NewResourceInstance() *ResourceInstance {
	return &ResourceInstance{
		Deposed: map[DeposedKey]*ResourceInstanceObject{},
	}
}

// HasCurrent returns true if this resource instance has a "current"-generation
// object. Most instances do, but this can briefly be false during a
// create-before-destroy replace operation when the current has been deposed
// but its replacement has not yet been created.
func (i *ResourceInstance) HasCurrent() bool {
	return i != nil && i.Current != nil
}

// HasDeposed returns true if this resource instance has a deposed object
// with the given key.
func (i *ResourceInstance) HasDeposed(key DeposedKey) bool {
	return i != nil && i.Deposed[key] != nil
}

// HasAnyDeposed returns true if this resource instance has one or more
// deposed objects.
func (i *ResourceInstance) HasAnyDeposed() bool {
	return i != nil && len(i.Deposed) > 0
}

// HasObjects returns true if this resource has any objects at all, whether
// current or deposed.
func (i *ResourceInstance) HasObjects() bool {
	return i.Current != nil || len(i.Deposed) != 0
}

// DeposeCurrentObject moves the current generation object, if present, into
// the deposed set. After this method returns, the instance has no current
// object.
//
// The return value is either the newly-allocated deposed key, or NotDeposed
// if the instance is already lacking a current instance object.
func (i *ResourceInstance) DeposeCurrentObject() DeposedKey {
	if !i.HasCurrent() {
		return NotDeposed
	}

	key := i.findUnusedDeposedKey()
	i.Deposed[key] = i.Current
	i.Current = nil
	return key
}

// GetGeneration retrieves the object of the given generation from the
// ResourceInstance, or returns nil if there is no such object.
//
// If the given generation is nil or invalid, this method will panic.
func (i *ResourceInstance) GetGeneration(gen Generation) *ResourceInstanceObject {
	if gen == CurrentGen {
		return i.Current
	}
	if dk, ok := gen.(DeposedKey); ok {
		return i.Deposed[dk]
	}
	if gen == nil {
		panic(fmt.Sprintf("get with nil Generation"))
	}
	// Should never fall out here, since the above covers all possible
	// Generation values.
	panic(fmt.Sprintf("get invalid Generation %#v", gen))
}

// findUnusedDeposedKey generates a unique DeposedKey that is guaranteed not to
// already be in use for this instance.
func (i *ResourceInstance) findUnusedDeposedKey() DeposedKey {
	for {
		key := NewDeposedKey()
		if _, exists := i.Deposed[key]; !exists {
			return key
		}
		// Spin until we find a unique one. This shouldn't take long, because
		// we have a 32-bit keyspace and there's rarely more than one deposed
		// instance.
	}
}

// EachMode specifies the multi-instance mode for a resource.
type EachMode rune

const (
	NoEach   EachMode = 0
	EachList EachMode = 'L'
	EachMap  EachMode = 'M'
)

//go:generate stringer -type EachMode

func eachModeForInstanceKey(key addrs.InstanceKey) EachMode {
	switch key.(type) {
	case addrs.IntKey:
		return EachList
	case addrs.StringKey:
		return EachMap
	default:
		if key == addrs.NoKey {
			return NoEach
		}
		panic(fmt.Sprintf("don't know an each mode for instance key %#v", key))
	}
}

// DeposedKey is a 8-character hex string used to uniquely identify deposed
// instance objects in the state.
type DeposedKey string

// NotDeposed is a special invalid value of DeposedKey that is used to represent
// the absense of a deposed key. It must not be used as an actual deposed key.
const NotDeposed = DeposedKey("")

var deposedKeyRand = rand.New(rand.NewSource(time.Now().UnixNano()))

// NewDeposedKey generates a pseudo-random deposed key. Because of the short
// length of these keys, uniqueness is not a natural consequence and so the
// caller should test to see if the generated key is already in use and generate
// another if so, until a unique key is found.
func NewDeposedKey() DeposedKey {
	v := deposedKeyRand.Uint32()
	return DeposedKey(fmt.Sprintf("%08x", v))
}

func (k DeposedKey) String() string {
	return string(k)
}

func (k DeposedKey) GoString() string {
	ks := string(k)
	switch {
	case ks == "":
		return "states.NotDeposed"
	default:
		return fmt.Sprintf("states.DeposedKey(%s)", ks)
	}
}

// generation is an implementation of Generation.
func (k DeposedKey) generation() {}
