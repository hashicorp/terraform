package versions

type setReleased struct{}

func (s setReleased) Has(v Version) bool {
	return v.Prerelease == ""
}

func (s setReleased) AllRequested() Set {
	// The set of all released versions requests nothing.
	return None
}

func (s setReleased) GoString() string {
	return "versions.Released"
}

// Released is a set containing all versions that have an empty prerelease
// string.
var Released Set

// Prerelease is a set containing all versions that have a prerelease marker.
// This is the complement of Released, or in other words it is
// All.Subtract(Released).
var Prerelease Set

func init() {
	Released = Set{setI: setReleased{}}
	Prerelease = All.Subtract(Released)
}
