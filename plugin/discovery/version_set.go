package discovery

import (
	version "github.com/hashicorp/go-version"
)

// A ConstraintStr is a string containing a possibly-invalid representation
// of a version constraint provided in configuration. Call Parse on it to
// obtain a real Constraint object, or discover that it is invalid.
type ConstraintStr string

// Parse transforms a ConstraintStr into a VersionSet if it is
// syntactically valid. If it isn't then an error is returned instead.
func (s ConstraintStr) Parse() (VersionSet, error) {
	raw, err := version.NewConstraint(string(s))
	if err != nil {
		return VersionSet{}, err
	}
	return VersionSet{raw}, nil
}

// MustParse is like Parse but it panics if the constraint string is invalid.
func (s ConstraintStr) MustParse() VersionSet {
	ret, err := s.Parse()
	if err != nil {
		panic(err)
	}
	return ret
}

// VersionSet represents a set of versions which any given Version is either
// a member of or not.
type VersionSet struct {
	// Internally a version set is actually a list of constraints that
	// *remove* versions from the set. Thus a VersionSet with an empty
	// Constraints list would be one that contains *all* versions.
	raw version.Constraints
}

// Has returns true if the given version is in the receiving set.
func (s VersionSet) Has(v Version) bool {
	return s.raw.Check(v.raw)
}

// Intersection combines the receving set with the given other set to produce a
// set that is the intersection of both sets, which is to say that it contains
// only the versions that are members of both sets.
func (s VersionSet) Intersection(other VersionSet) VersionSet {
	raw := make(version.Constraints, 0, len(s.raw)+len(other.raw))

	// Since "raw" is a list of constraints that remove versions from the set,
	// "Intersection" is implemented by concatenating together those lists,
	// thus leaving behind only the versions not removed by either list.
	raw = append(raw, s.raw...)
	raw = append(raw, other.raw...)

	return VersionSet{raw}
}

// String returns a string representation of the set members as a set
// of range constraints.
func (s VersionSet) String() string {
	return s.raw.String()
}
