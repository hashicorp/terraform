package resource

type NotFoundError struct {
	LastError    error
	LastRequest  interface{}
	LastResponse interface{}
	Message      string
	Retries      int
}

func (e *NotFoundError) Error() string {
	if e.Message != "" {
		return e.Message
	}

	return "couldn't find resource"
}

func NewNotFoundError(err string) *NotFoundError {
	return &NotFoundError{Message: err}
}
