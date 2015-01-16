package schema

// Equal is an interface that checks for deep equality between two objects.
type Equal interface {
	Equal(interface{}) bool
}
