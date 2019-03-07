package terraform

import "context"

// UIInput is the interface that must be implemented to ask for input
// from this user. This should forward the request to wherever the user
// inputs things to ask for values.
type UIInput interface {
	Input(context.Context, *InputOpts) (string, error)
}

// InputOpts are options for asking for input.
type InputOpts struct {
	// Id is a unique ID for the question being asked that might be
	// used for logging or to look up a prior answered question.
	Id string

	// Query is a human-friendly question for inputting this value.
	Query string

	// Description is a description about what this option is. Be wary
	// that this will probably be in a terminal so split lines as you see
	// necessary.
	Description string

	// Default will be the value returned if no data is entered.
	Default string
}
