package pgacl

// ACL is a generic interface that all pgacl types must adhere to
type ACL interface {
	// String creates a PostgreSQL compatible ACL string
	String() string
}
