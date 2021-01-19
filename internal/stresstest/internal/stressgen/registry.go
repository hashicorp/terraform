package stressgen

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// Registry is a container for data to help us to randomly generate valid
// references between objects, and to ensure those references will
// remain valid under randomly-generated modifications.
//
// While we try to keep the ConfigObjects in a randomly-generated configuration
// as self-contained as possible, a lot of Terraform behaviors only emerge
// as a result of references between objects and so we need to be able to
// randomly generate those too. This type is here to coordinate that.
//
// Each randomly-generated module has its own Registry, because each Terraform
// module has its own separate namespace. Mirroring the usual configuration
// structure, each randomly-generated configuration has one root Registry and
// then an additional Registry for each of the child modules it calls.
type Registry struct {
	// Parent and Children together represent the tree of registries, which
	// mirrors the tree of module instances described by the generated
	// configuration.
	Parent   *Registry
	Children map[addrs.ModuleInstanceStep]*Registry

	// ModuleAddr is the address of the module instance that this registry
	// belongs to.
	ModuleAddr addrs.ModuleInstance

	// RefValues tracks the values we expect to see appear for particular
	// referenceable objects during the plan or apply steps.
	// ConfigObject implementations should make sure that any reference target
	// they added to the Namespace gets a corresponding RefValue entry during
	// Instantiate, or downstream config building will panic on trying to
	// resolve those references.
	//
	// The keys of RefValues are the string representations of
	// addrs.Referencable values. There are some methods of Registry providing
	// a more ergonomic and type-checked interface to updating and reading
	// this map.
	RefValues map[string]cty.Value

	// VariableValues tracks values for the module's input variables that are
	// set by the caller. This doesn't include variables that the caller leaves
	// unset in order to accept the defaults.
	//
	// VariableValues is a tricky case because the sequence of events for
	// handling input variables requires careful coordination between
	// the generator of the caller and the generator of the called module:
	// for each input variable that the generator declares for the child
	// module, the generator for the caller must create any necessary entries
	// in here (using RegisterVariableValue) _before_ calling Instance on
	// the input variable, so that the input variable instantiation logic can
	// look here (with VariableValue) to determine which value it should
	// expect.
	VariableValues map[addrs.InputVariable]cty.Value
}

// NewRootRegistry creates and returns an empty registry that has no parent
// registry.
func NewRootRegistry() *Registry {
	return &Registry{
		ModuleAddr:     addrs.RootModuleInstance,
		Children:       make(map[addrs.ModuleInstanceStep]*Registry),
		RefValues:      make(map[string]cty.Value),
		VariableValues: make(map[addrs.InputVariable]cty.Value),
	}
}

// NewChild creates and returns an empty registry that is registered as a child
// of the reciever.
//
// The given name must be unique within the space of child registry names in
// the reciever, or this function will panic. The name of a child registry
// should match the name of the module call that implied its existence.
func (r *Registry) NewChild(modStep addrs.ModuleInstanceStep) *Registry {
	if _, exists := r.Children[modStep]; exists {
		panic(fmt.Sprintf("registry already has a child module instance %q", modStep))
	}
	ret := NewRootRegistry()
	ret.Parent = r
	ret.ModuleAddr = r.ModuleAddr.Child(modStep.Name, modStep.InstanceKey)
	return ret
}

// RegisterRefValue records the expected value of a particular referencable
// object that was previously declared in the namespace corresponding to this
// registry.
//
// Note that although references can be to attributes or indices under a
// referencable object, we always register entire objects here and then let
// the RefValue method be responsible for retrieving a specific sub-element
// if needed.
//
// Each object may be registered only once. If a registry recieves two calls
// to register the same object, this method will panic.
func (r *Registry) RegisterRefValue(objAddr addrs.Referenceable, v cty.Value) {
	k := objAddr.String()
	if _, exists := r.RefValues[k]; exists {
		panic(fmt.Sprintf("duplicate registration of value for %s", k))
	}
	r.RefValues[k] = v
}

// RefValue consults the RefValues table to find the expected value for a
// particular reference expression.
//
// This function will panic if asked to access something that hasn't been
// registered in the RefValues table, because that suggests a bug in the
// Instantiate method of whichever ConfigObject declared that reference to
// be allowed in the first place.
func (r *Registry) RefValue(objAddr addrs.Referenceable, path cty.Path) cty.Value {
	val, exists := r.RefValues[objAddr.String()]
	if !exists {
		panic(fmt.Sprintf("no expected value has been registered for %s", objAddr.String()))
	}
	val, err := path.Apply(val)
	if err != nil {
		panic(fmt.Sprintf("expected value for %s doesn't support %s: %s", objAddr.String(), tfdiags.FormatCtyPath(path), err))
	}
	return val
}

// RegisterVariableValue is used by the function generating the caller of
// a module in order to tell the variable objects in the called module which
// values they should expect to be passed.
//
// This should be called for all variables that have CallerWillSet set to true,
// and not called at all for variables where CallerWillSet is false.
func (r *Registry) RegisterVariableValue(addr addrs.InputVariable, v cty.Value) {
	if _, exists := r.VariableValues[addr]; exists {
		panic(fmt.Sprintf("duplicate registration of caller-set value for %s", addr))
	}
	r.VariableValues[addr] = v
}

// VariableValue returns the caller-set value for a particular variable, but
// only if that variable was generated with CallerWillSet set to true.
// If no value is registered for the given variable then this function will
// panic, suggesting either a bug in the generator for the calling module
// (failing to register a caller-set variable) or in the generator for the
// called module (trying to retrieve a value for a variable that isn't
// caller-set.)
func (r *Registry) VariableValue(addr addrs.InputVariable) cty.Value {
	ret, exists := r.VariableValues[addr]
	if !exists {
		panic(fmt.Sprintf("no caller-set value for %s", addr))
	}
	return ret
}
