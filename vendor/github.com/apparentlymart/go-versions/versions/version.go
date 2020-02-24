package versions

import (
	"fmt"
	"strings"
)

// Version represents a single version.
type Version struct {
	Major      uint64
	Minor      uint64
	Patch      uint64
	Prerelease VersionExtra
	Metadata   VersionExtra
}

// Unspecified is the zero value of Version and represents the absense of a
// version number.
//
// Note that this is indistinguishable from the explicit version that
// results from parsing the string "0.0.0".
var Unspecified Version

// Same returns true if the receiver has the same precedence as the other
// given version. In other words, it has the same major, minor and patch
// version number and an identical prerelease portion. The Metadata, if
// any, is not considered.
func (v Version) Same(other Version) bool {
	return (v.Major == other.Major &&
		v.Minor == other.Minor &&
		v.Patch == other.Patch &&
		v.Prerelease == other.Prerelease)
}

// Comparable returns a version that is the same as the receiver but its
// metadata is the empty string. For Comparable versions, the standard
// equality operator == is equivalent to method Same.
func (v Version) Comparable() Version {
	v.Metadata = ""
	return v
}

// String is an implementation of fmt.Stringer that returns the receiver
// in the canonical "semver" format.
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s = fmt.Sprintf("%s-%s", s, v.Prerelease)
	}
	if v.Metadata != "" {
		s = fmt.Sprintf("%s+%s", s, v.Metadata)
	}
	return s
}

func (v Version) GoString() string {
	return fmt.Sprintf("versions.MustParseVersion(%q)", v.String())
}

// LessThan returns true if the receiver has a lower precedence than the
// other given version, as defined by the semantic versioning specification.
func (v Version) LessThan(other Version) bool {
	switch {
	case v.Major != other.Major:
		return v.Major < other.Major
	case v.Minor != other.Minor:
		return v.Minor < other.Minor
	case v.Patch != other.Patch:
		return v.Patch < other.Patch
	case v.Prerelease != other.Prerelease:
		if v.Prerelease == "" {
			return false
		}
		if other.Prerelease == "" {
			return true
		}
		return v.Prerelease.LessThan(other.Prerelease)
	default:
		return false
	}
}

// GreaterThan returns true if the receiver has a higher precedence than the
// other given version, as defined by the semantic versioning specification.
func (v Version) GreaterThan(other Version) bool {
	switch {
	case v.Major != other.Major:
		return v.Major > other.Major
	case v.Minor != other.Minor:
		return v.Minor > other.Minor
	case v.Patch != other.Patch:
		return v.Patch > other.Patch
	case v.Prerelease != other.Prerelease:
		if v.Prerelease == "" {
			return true
		}
		if other.Prerelease == "" {
			return false
		}
		return !v.Prerelease.LessThan(other.Prerelease)
	default:
		return false
	}
}

// MarshalText is an implementation of encoding.TextMarshaler, allowing versions
// to be automatically marshalled for text-based serialization formats,
// including encoding/json.
//
// The format used is that returned by String, which can be parsed using
// ParseVersion.
func (v Version) MarshalText() (text []byte, err error) {
	return []byte(v.String()), nil
}

// UnmarshalText is an implementation of encoding.TextUnmarshaler, allowing
// versions to be automatically unmarshalled from strings in text-based
// serialization formats, including encoding/json.
//
// The format expected is what is accepted by ParseVersion. Any parser errors
// are passed on verbatim to the caller.
func (v *Version) UnmarshalText(text []byte) error {
	str := string(text)
	new, err := ParseVersion(str)
	if err != nil {
		return err
	}
	*v = new
	return nil
}

// VersionExtra represents a string containing dot-delimited tokens, as used
// in the pre-release and build metadata portions of a Semantic Versioning
// version expression.
type VersionExtra string

// Parts tokenizes the string into its separate parts by splitting on dots.
//
// The result is undefined if the receiver is not valid per the semver spec,
func (e VersionExtra) Parts() []string {
	return strings.Split(string(e), ".")
}

func (e VersionExtra) Raw() string {
	return string(e)
}

// LessThan returns true if the receiever has lower precedence than the
// other given VersionExtra string, per the rules defined in the semver
// spec for pre-release versions.
//
// Build metadata has no defined precedence rules, so it is not meaningful
// to call this method on a VersionExtra representing build metadata.
func (e VersionExtra) LessThan(other VersionExtra) bool {
	if e == other {
		// Easy path
		return false
	}

	s1 := string(e)
	s2 := string(other)
	for {
		d1 := strings.IndexByte(s1, '.')
		d2 := strings.IndexByte(s2, '.')

		switch {
		case d1 == -1 && d2 != -1:
			// s1 has fewer parts, so it precedes s2
			return true
		case d2 == -1 && d1 != -1:
			// s1 has more parts, so it succeeds s2
			return false
		case d1 == -1: // d2 must be -1 too, because of the above
			// this is our last portion to compare
			return lessThanStr(s1, s2)
		default:
			s1s := s1[:d1]
			s2s := s2[:d2]
			if s1s != s2s {
				return lessThanStr(s1s, s2s)
			}
			s1 = s1[d1+1:]
			s2 = s2[d2+1:]
		}
	}
}

func lessThanStr(s1, s2 string) bool {
	// How we compare here depends on whether the string is entirely consistent of digits
	s1Numeric := true
	s2Numeric := true
	for _, c := range s1 {
		if c < '0' || c > '9' {
			s1Numeric = false
			break
		}
	}
	for _, c := range s2 {
		if c < '0' || c > '9' {
			s2Numeric = false
			break
		}
	}

	switch {
	case s1Numeric && !s2Numeric:
		return true
	case s2Numeric && !s1Numeric:
		return false
	case s1Numeric: // s2Numeric must also be true
		switch {
		case len(s1) < len(s2):
			return true
		case len(s2) < len(s1):
			return false
		default:
			return s1 < s2
		}
	default:
		return s1 < s2
	}
}
