package versions

// Set is a set of versions, usually created by parsing a constraint string.
type Set struct {
	setI
}

// setI is the private interface implemented by our various constraint
// operators.
type setI interface {
	Has(v Version) bool
	AllRequested() Set
	GoString() string
}

// Has returns true if the given version is a member of the receiving set.
func (s Set) Has(v Version) bool {
	// The special Unspecified version is excluded as soon as any sort of
	// constraint is applied, and so the only set it is a member of is
	// the special All set.
	if v == Unspecified {
		return s == All
	}

	return s.setI.Has(v)
}

// Requests returns true if the given version is specifically requested by
// the receiving set.
//
// Requesting is a stronger form of set membership that represents an explicit
// request for a particular version, as opposed to the version just happening
// to match some criteria.
//
// The functions Only and Selection mark their arguments as requested in
// their returned sets. Exact version constraints given in constraint strings
// also mark their versions as requested.
//
// The concept of requesting is intended to help deal with pre-release versions
// in a safe and convenient way. When given generic version constraints like
// ">= 1.0.0" the user generally does not intend to match a pre-release version
// like "2.0.0-beta1", but it is important to stil be able to use that
// version if explicitly requested using the constraint string "2.0.0-beta1".
func (s Set) Requests(v Version) bool {
	return s.AllRequested().Has(v)
}

// AllRequested returns a subset of the receiver containing only the requested
// versions, as defined in the documentation for the method Requests.
//
// This can be used in conjunction with the predefined set "Released" to
// include pre-release versions only by explicit request, which is supported
// via the helper method WithoutUnrequestedPrereleases.
//
// The result of AllRequested is always a finite set.
func (s Set) AllRequested() Set {
	return s.setI.AllRequested()
}

// WithoutUnrequestedPrereleases returns a new set that includes all released
// versions from the receiving set, plus any explicitly-requested pre-releases,
// but does not include any unrequested pre-releases.
//
// "Requested" here is as defined in the documentation for the "Requests" method.
//
// This method is equivalent to the following set operations:
//
//     versions.Union(s.AllRequested(), s.Intersection(versions.Released))
func (s Set) WithoutUnrequestedPrereleases() Set {
	return Union(s.AllRequested(), Released.Intersection(s))
}

// UnmarshalText is an implementation of encoding.TextUnmarshaler, allowing
// sets to be automatically unmarshalled from strings in text-based
// serialization formats, including encoding/json.
//
// The format expected is what is accepted by MeetingConstraintsString. Any
// parser errors are passed on verbatim to the caller.
func (s *Set) UnmarshalText(text []byte) error {
	str := string(text)
	new, err := MeetingConstraintsString(str)
	if err != nil {
		return err
	}
	*s = new
	return nil
}

var InitialDevelopment Set = OlderThan(MustParseVersion("1.0.0"))
