package jsonstate

import "encoding/json"

// expression represents any unparsed expression
type expression struct {
	// "constant_value" is set only if the expression contains no references to
	// other objects, in which case it gives the resulting constant value. This
	// is mapped as for the individual values in the common value
	// representation.
	ConstantValue json.RawMessage `json:"constant_value,omitempty"`

	// Alternatively, "references" will be set to a list of references in the
	// expression. Multi-step references will be unwrapped and duplicated for
	// each significant traversal step, allowing callers to more easily
	// recognize the objects they care about without attempting to parse the
	// expressions. Callers should only use string equality checks here, since
	// the syntax may be extended in future releases.
	References []string `json:"references,omitempty"`

	// "source" is an object describing the source span of this expression in
	// the configuration. Callers might use this, for example, to extract a raw
	// source code snippet for display purposes.
	Source source `json:"source"`
}
