package seelog

// Base struct for custom errors.
type baseError struct {
	message string
}

func (be baseError) Error() string {
	return be.message
}
