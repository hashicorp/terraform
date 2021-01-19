package stressgen

import (
	"fmt"
	"math/rand"

	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

// Namespace is an object used for coordination between multiple different
// generators, to allow different objects to refer to each other while
// making sure the resulting module is valid.
//
// In a sense Namespace is a generation-time analog to Registry. Namespace
// tracks the static declarations of items in a module, and then Registry
// tracks the dynamic values associated with those declarations on a
// per-module-instance basis.
//
// Namespace has some mutation methods which are used during the generation
// process, but external callers should not use these and should treat a
// Namespace as immutable once its associated configuration has been generated.
type Namespace struct {
	ModuleAddr addrs.Module

	// issuedNames is where we track which random names we already generated,
	// so we can guarantee not to issue the same one twice in the same module.
	//
	// This is technically more conservative than it actually needs to be: it
	// is valid for two objects of different types to share a name, for example,
	// but pretending that we have a flat namespace makes things simpler here
	// because we don't need to have separate tables for each object type.
	issuedNames map[string]struct{}

	// RefTargets are reference expressions that are announced by one object
	// so that another object downstream might choose to refer to it.
	//
	// This slice is in no particular order, and registring a new target
	// just appends to the end of it. Instead of accessing this field directly,
	// use the DeclareReferenceable and GenerateReference methods.
	RefTargets []*ConfigExprRef

	// OutputValues tracks the output values that were generated for the module
	// represented by this namespace. The configuration generator registers
	// outputs here as they are generated, so that a calling module can
	// potentially then refer to them.
	OutputValues map[string]*ConfigOutput
}

// NewNamespace creates and returns an empty namespace ready to be populated.
func NewNamespace() *Namespace {
	return &Namespace{
		ModuleAddr:   addrs.RootModule,
		issuedNames:  make(map[string]struct{}),
		OutputValues: make(map[string]*ConfigOutput),
	}
}

// ChildNamespace creates and returns an empty namespace which represents a
// child module of the recieving namespace, with the given call name.
func (n *Namespace) ChildNamespace(callName string) *Namespace {
	ret := NewNamespace()
	ret.ModuleAddr = n.ModuleAddr.Child(callName)
	return ret
}

// GenerateShortName is like the package-level function of the same name,
// except that the reciever remembers names it has returned before and
// guarantees not to return the same string twice.
func (n *Namespace) GenerateShortName(rnd *rand.Rand) string {
	for {
		ret := GenerateShortName(rnd)
		if _, exists := n.issuedNames[ret]; !exists {
			n.issuedNames[ret] = struct{}{}
			return ret
		}
	}
}

// GenerateShortModifierName is like the package-level function of the same name,
// except that the reciever remembers names it has returned before and
// guarantees not to return the same string twice.
func (n *Namespace) GenerateShortModifierName(rnd *rand.Rand) string {
	for {
		ret := GenerateShortModifierName(rnd)
		if _, exists := n.issuedNames[ret]; !exists {
			n.issuedNames[ret] = struct{}{}
			return ret
		}
	}
}

// GenerateLongName generates a "long" unique name string that contains a
// series of words separated by dashes. These might be useful as unique
// names for remote objects in the fake providers.
//
// By convention we typically use GenerateShortName for names used within
// the Terraform language itself, such as variable names, but use
// GenerateLongName for names sent to the "remote systems" represented by
// our fake providers, just to make those two cases a bit more distinct
// for folks reviewing a dense randomly-generated configuration.
//
// Currently this function generates opinions about animals, although that's
// an implementation detail subject to change in future.
func (n *Namespace) GenerateLongName(rnd *rand.Rand) string {
	for {
		ret := GenerateLongString(rnd)
		if _, exists := n.issuedNames[ret]; !exists {
			n.issuedNames[ret] = struct{}{}
			return ret
		}
	}
}

// DeclareOutputValue can be called by an object generator to notify the
// call of the module represented by the current namespace that there's an
// output value available to refer to.
//
// Each distinct output value name may only be declared once. If two calls
// have the same name then this method will panic.
func (n *Namespace) DeclareOutputValue(o *ConfigOutput) {
	if _, exists := n.OutputValues[o.Addr.Name]; exists {
		panic(fmt.Sprintf("duplicate declaration of output value %q", o.Addr.Name))
	}
	n.OutputValues[o.Addr.Name] = o
}

// DeclareReferenceable can be called by an object generator to notify
// downstream objects that it has contributed something to the reference
// symbol table which could be used in an expression.
//
// If the generator for an object registers an object with this method then it
// must also register an expected value for that object in each Registry
// passed to that object's Instantiate method, or else downstream object
// construction will panic.
func (n *Namespace) DeclareReferenceable(expr *ConfigExprRef) {
	n.RefTargets = append(n.RefTargets, expr)
}

// GenerateExpression uses the given random number generator to randomly
// generate an expression, which might be either a constant expression or
// a reference expression.
//
// This function slightly favors returning references, in order to slightly
// encourage generating interestingly-connected dependency graphs.
func (n *Namespace) GenerateExpression(rnd *rand.Rand) ConfigExpr {
	if len(n.RefTargets) == 0 {
		// If no ref targets are registered yet, we don't have any option
		// but to return a constant.
		return n.GenerateConstStringExpr(rnd)
	}
	useRef := decideBool(rnd, 55)
	if useRef {
		return n.GenerateReference(rnd)
	}
	return n.GenerateConstStringExpr(rnd)
}

// GenerateReference uses the given random number generator to select one of
// the reference targets registered by earlier calls to DeclareReferenceable.
//
// This function will return nil if nothing has been registered yet.
func (n *Namespace) GenerateReference(rnd *rand.Rand) *ConfigExprRef {
	if len(n.RefTargets) == 0 {
		return nil
	}
	idx := rand.Intn(len(n.RefTargets))
	return n.RefTargets[idx]
}

// GenerateConstStringExpr uses the given random number generator to generate a
// random expression with a constant string value.
//
// Currently this function uses strings generated by GenerateLongName to ensure
// that the results will be unique across an entire module and thus that the
// values could potentially be used to populate unique identifiers.
func (n *Namespace) GenerateConstStringExpr(rnd *rand.Rand) *ConfigExprConst {
	str := n.GenerateLongName(rnd)
	return &ConfigExprConst{cty.StringVal(str)}
}
