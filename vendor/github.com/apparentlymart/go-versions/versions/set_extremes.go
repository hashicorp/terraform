package versions

// All is an infinite set containing all possible versions.
var All Set

// None is a finite set containing no versions.
var None Set

type setExtreme bool

func (s setExtreme) Has(v Version) bool {
	return bool(s)
}

func (s setExtreme) AllRequested() Set {
	// The extreme sets request nothing.
	return None
}

func (s setExtreme) GoString() string {
	switch bool(s) {
	case true:
		return "versions.All"
	case false:
		return "versions.None"
	default:
		panic("strange new boolean value")
	}
}

var _ setFinite = setExtreme(false)

func (s setExtreme) isFinite() bool {
	// Only None is finite
	return !bool(s)
}

func (s setExtreme) listVersions() List {
	return nil
}

func init() {
	All = Set{
		setI: setExtreme(true),
	}
	None = Set{
		setI: setExtreme(false),
	}
}
