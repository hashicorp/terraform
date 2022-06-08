package jsonplan

// conditionResult is the representation of an evaluated condition block.
type conditionResult struct {
	// checkAddress is the globally-unique address of the condition block. This
	// is intentionally unexported as it is an implementation detail.
	checkAddress string

	// Address is the absolute address of the condition's containing object.
	Address string `json:"address,omitempty"`

	// Type is the condition block type, and is one of ResourcePrecondition,
	// ResourcePostcondition, or OutputPrecondition.
	Type string `json:"condition_type,omitempty"`

	// Result is true if the condition succeeds, and false if it fails or is
	// known only at apply time.
	Result bool `json:"result"`

	// Unknown is true if the condition can only be evaluated at apply time.
	Unknown bool `json:"unknown"`

	// ErrorMessage is the custom error for a failing condition. It is only
	// present if the condition fails.
	ErrorMessage string `json:"error_message,omitempty"`
}
