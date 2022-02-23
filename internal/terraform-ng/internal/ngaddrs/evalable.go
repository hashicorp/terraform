package ngaddrs

type Evalable interface {
	// HACK: Since we currently have this split between addrs types and ngaddrs
	// types, we can't define a Referenceable which can span across both
	// packages while making it a closed set of implementations as we normally
	// would. Instead, this interface matches far more address types than it
	// really ought to, which is fine while we're prototyping but means we
	// must be careful not to assign inappropriate values to variables of
	// this type: the compiler won't check us.

	// String produces a string representation of the address that could be
	// parsed as a HCL traversal and passed to ParseRef to produce an identical
	// result.
	String() string
}

// The following are all of the address types we intend to be assignable
// to "Evalable". Although the Go compiler will allow others, using them
// is always incorrect.
var (
	_ Evalable = AbsComponentCall{}
	_ Evalable = AbsComponentGroupCall{}
)
