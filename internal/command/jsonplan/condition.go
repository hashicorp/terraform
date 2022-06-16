package jsonplan

// conditionResult is the representation of an evaluated condition block.
//
// This no longer really fits how Terraform is modelling checks -- we're now
// treating check status as a whole-object thing rather than an individual
// condition thing -- but we've preserved this for now to remain as compatible
// as possible with the interface we'd documented as part of the Terraform v1.2
// release, before we'd really solidified the use-cases for checks outside
// of just making a single plan and apply operation fail with an error.
type conditionResult struct {
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
