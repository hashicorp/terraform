package terraform

// EvalContext is the interface that is given to eval nodes to execute.
type EvalContext interface {
	// InitProvider initializes the provider with the given name and
	// returns the implementation of the resource provider or an error.
	InitProvider(string) (ResourceProvider, error)

	// Provider gets the provider instance with the given name (already
	// initialized) or returns nil if the provider isn't initialized.
	Provider(string) ResourceProvider
}

// EvalNode is the interface that must be implemented by graph nodes to
// evaluate/execute.
type EvalNode interface {
	// Args returns the arguments for this node as well as the list of
	// expected types. The expected types are only used for type checking
	// and not used at runtime.
	Args() ([]EvalNode, []EvalType)

	// Eval evaluates this node with the given context.
	Eval(EvalContext) (interface{}, error)

	// Type returns the type that will be returned by this node.
	Type() EvalType
}

// GraphNodeEvalable is the interface that graph nodes must implement
// to enable valuation.
type GraphNodeEvalable interface {
	EvalTree() EvalNode
}
