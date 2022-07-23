package jsonplan

// conditionResult is the representation of an evaluated condition block.
//
// This no longer really fits how Terraform is modelling checks -- we're now
// treating check status as a whole-object thing rather than an individual
// condition thing -- but we've preserved this for now to remain as compatible
// as possible with the interface we'd experimentally-implemented but not
// documented in the Terraform v1.2 release, before we'd really solidified the
// use-cases for checks outside of just making a single plan and apply
// operation fail with an error.
type conditionResult struct {
	// This is a weird "pseudo-comment" noting that we're deprecating this
	// not-previously-documented, experimental representation of conditions
	// in favor of the "checks" property which better fits Terraform Core's
	// modelling of checks.
	DeprecationNotice conditionResultDeprecationNotice `json:"//"`

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

type conditionResultDeprecationNotice struct{}

func (n conditionResultDeprecationNotice) MarshalJSON() ([]byte, error) {
	return conditionResultDeprecationNoticeJSON, nil
}

var conditionResultDeprecationNoticeJSON = []byte(`"This previously-experimental representation of conditions is deprecated and will be removed in Terraform v1.4. Use the 'checks' property instead."`)
