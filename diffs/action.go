package diffs

type Action rune

//go:generate stringer -type Action

const (
	// NoAction is the zero value of type Action.
	//
	// NoAction is not usually used as a change type, but might be used as
	// one in a []*Change that describes modifications to a sequence of
	// values, in order to serve as context.
	NoAction Action = 0

	// Create is an action that represents creating a new remote object where no
	// object existed before.
	Create Action = '+'

	// Read is an action that represents retrieving data from a remote object
	// without modifying it.
	Read Action = '^'

	// Update is an action that represents an in-place change to an existing
	// remote object.
	Update Action = '~'

	// Delete is an action that represents destroying an existing remote object.
	Delete Action = '-'

	// Replace is an action that represents destroying an object and creating
	// a new object in its place.
	//
	// The destroy does not necessarily precede the create, if the object in
	// question uses the "create before destroy" lifecycle.
	Replace Action = '±'
)
