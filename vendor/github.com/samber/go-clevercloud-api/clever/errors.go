package clever

type BadRequestError struct{ body string }
type ForbiddenError struct{ body string }
type NotFoundError struct{ body string }
type InternalServerError struct{ body string }

func (err BadRequestError) Error() string {
	return "Bad request (400) => " + err.body
}
func (err ForbiddenError) Error() string {
	return "Forbidden (403) => " + err.body
}
func (err NotFoundError) Error() string {
	return "Not found (404) => " + err.body
}
func (err InternalServerError) Error() string {
	return "Internal server error (500) => " + err.body
}
