package stackconfig

// Config represents a node in a tree of stacks that are to be planned and
// applied together.
//
// A fully-resolved stack configuration has a root node of this type, which
// can have zero or more child nodes that are also of this type, and so on
// to arbitrary levels of nesting.
type Config struct {
	// Stack is the definition of this node in the stack tree.
	Stack *Stack

	// Children describes all of the embedded stacks nested directly beneath
	// this node in the stack tree. The keys match the labels on the "stack"
	// blocks in the configuration that [Config.Stack] was built from, and
	// so also match the keys in the EmbeddedStacks field of that Stack.
	Children map[string]*Stack
}
