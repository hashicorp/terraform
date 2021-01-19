package stressgen

import (
	"math/rand"

	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/hashicorp/terraform/states"
)

// ConfigObject is an interface implemented by types representing items that
// can be included in a generated test configuration.
//
// Each ConfigObject typically represents one configuration block in a module,
// and has a few different responsibilities. The most important is to generate
// the actual configuration block for the object, but a ConfigObject can be
// made more useful by providing a verifier that checks whether the final
// state matches the goal of the configuration, and by registering objects that
// it makes available in the symbol table that later-constructed objects might
// potentially refer to in order to create a proper dependency graph that is
// more likely to detect race conditions.
//
// Some ConfigObject implementations can contain other nested ConfigObject
// implementations. For example, ModuleCall contains another whole module
// which its parent module will call.
type ConfigObject interface {
	// DisplayName returns a reasonable identifier for this object to use in
	// UI output. It's not necessarily unique across a whole configuration, but
	// should be as unique as possible. For objects that already have
	// conventional relative address syntax (which includes anything that
	// can be used in a value expression), that relative address is a good
	// choice.
	DisplayName() string

	// AppendConfig appends one or more configuration blocks to the given
	// body, which represents the top-level body of a .tf file.
	AppendConfig(to *hclwrite.Body)

	// GenerateModified produces a new object that is in some sense similar to
	// the receiver but possibly modified based on results from the given
	// random number generator. It might also choose to return a nil
	// ConfigObject, which represents removing the object from the
	// configuration altogether.
	//
	// Since ConfigObject data is expected to be immutable once generated,
	// implementations of GenerateModified might return a value that shares
	// backing memory with the reciever in cases where a particular part
	// of the object hasn't been modified. It's also valid for CreateObject
	// to just return the original object directly, if no modifications are
	// actually needed.
	//
	// The result will typically have the same concrete type as the reciever,
	// but that's not actually required and so callers should not assume
	// type identity.
	GenerateModified(rnd *rand.Rand, ns *Namespace) ConfigObject

	// Instantiate creates a new instance of the reciever by associating it
	// with a particular registry. Instantiate can read the registry in order
	// to find the values of any external objects it refers to, and should
	// then also add new entries to the registry to record whatever
	// contributions it will itself make to the rest of the module it is
	// part of.
	//
	// A single object can be instantiated multiple times if its definition
	// exists in a child module that has either "for_each" or "count" set
	// in its call.
	Instantiate(reg *Registry) ConfigObjectInstance
}

// ConfigObjectInstance represents one of possibly several instances of a
// ConfigObject.
//
// We use this to deal with the fact that a particular configuration object
// can potentially be instantiated multiple times if it's declared inside a
// module which uses either the "for_each" or "count" meta-argument. In that
// case, all of the instances share the same configuration but the values
// expected in the state will differ.
type ConfigObjectInstance interface {
	// DisplayName returns a reasonable identifier for this object to use in
	// UI output. It's not necessarily unique across a whole configuration, but
	// should be as unique as possible. For objects that already have
	// conventional absolute address syntax, like resources and modules,
	// the string serialization of those addresses are a good choice.
	DisplayName() string

	// Object returns the ConfigObject that the reciever is an instance of.
	Object() ConfigObject

	// CheckState compares the relevant parts of the given state to the
	// original configuration for itself and returns one or more errors if
	// anything doesn't match expectations.
	//
	// CheckState also recieves the prior state, which is a snapshot of what
	// the state looked like at the end of the previous step in the config
	// series. The prior state will be empty if we're checking the result of
	// applying only the first step. Objects might use this information if they
	// are testing features that should cause Terraform to prefer a value
	// from the state even if the configuration doesn't match, such as
	// ignore_changes for resources.
	CheckState(prior, new *states.State) []error
}
