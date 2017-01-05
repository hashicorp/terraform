package errors

type UnauthorizedError struct {
	description string
}

func NewUnauthorizedError(description string) error {
	return &UnauthorizedError{description: description}
}

func (err *UnauthorizedError) Error() string {
	return "Unauthorized error: " + err.description
}
