package discovery

import (
	"sort"

	version "github.com/hashicorp/go-version"
)

// A ConstraintStr is a string containing a possibly-invalid representation
// of a version constraint provided in configuration. Call Parse on it to
// obtain a real Constraint object, or discover that it is invalid.
type ConstraintStr string

// Parse transforms a ConstraintStr into a Constraints if it is
// syntactically valid. If it isn't then an error is returned instead.
func (s ConstraintStr) Parse() (Constraints, error) {
	raw, err := version.NewConstraint(string(s))
	if err != nil {
		return Constraints{}, err
	}
	return Constraints{raw}, nil
}

// MustParse is like Parse but it panics if the constraint string is invalid.
func (s ConstraintStr) MustParse() Constraints {
	ret, err := s.Parse()
	if err != nil {
		panic(err)
	}
	return ret
}

// Constraints represents a set of versions which any given Version is either
// a member of or not.
type Constraints struct {
	raw version.Constraints
}

// NewConstraints creates a Constraints based on a version.Constraints.
func NewConstraints(c version.Constraints) Constraints {
	return Constraints{c}
}

// AllVersions is a Constraints containing all versions
var AllVersions Constraints

func init() {
	AllVersions = Constraints{
		raw: make(version.Constraints, 0),
	}
}

// Allows returns true if the given version permitted by the receiving
// constraints set.
func (s Constraints) Allows(v Version) bool {
	return s.raw.Check(v.raw)
}

// Append combines the receiving set with the given other set to produce
// a set that is the intersection of both sets, which is to say that resulting
// constraints contain only the versions that are members of both.
func (s Constraints) Append(other Constraints) Constraints {
	raw := make(version.Constraints, 0, len(s.raw)+len(other.raw))

	// Since "raw" is a list of constraints that remove versions from the set,
	// "Intersection" is implemented by concatenating together those lists,
	// thus leaving behind only the versions not removed by either list.
	raw = append(raw, s.raw...)
	raw = append(raw, other.raw...)

	// while the set is unordered, we sort these lexically for consistent output
	sort.Slice(raw, func(i, j int) bool {
		return raw[i].String() < raw[j].String()
	})

	return Constraints{raw}
}

// String returns a string representation of the set members as a set
// of range constraints.
func (s Constraints) String() string {
	return s.raw.String()
}

// Unconstrained returns true if and only if the receiver is an empty
// constraint set.
func (s Constraints) Unconstrained() bool {
	return len(s.raw) == 0
}
