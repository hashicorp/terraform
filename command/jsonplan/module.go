package jsonplan

// module is the representation of a module in state This can be the root module
// or a child module
type module struct {
	Resources []resource

	// Address is the absolute module address, omitted for the root module
	Address string `json:"address,omitempty"`

	// Each module object can optionally have its own nested "child_modules",
	// recursively describing the full module tree.
	ChildModules []module `json:"child_modules,omitempty"`
}

type moduleCall struct {
	ResolvedSource    string      `json:"resolved_source"`
	Expressions       expressions `json:"expressions"`
	CountExpression   expression  `json:"count_expression"`
	ForEachExpression expression  `json:"for_each_expression"`
	Module            module      `json:"module"`
}
