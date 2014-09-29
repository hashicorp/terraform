package terraform

// UIInput is the interface that must be implemented to ask for input
// from this user. This should forward the request to wherever the user
// inputs things to ask for values.
type UIInput interface {
	Input(*InputOpts) (string, error)
}

// InputOpts are options for asking for input.
type InputOpts struct {
	// Id is a unique ID for the question being asked that might be
	// used for logging or to look up a prior answered question.
	Id    string

	// Query is a human-friendly question for inputting this value.
	Query string
}
