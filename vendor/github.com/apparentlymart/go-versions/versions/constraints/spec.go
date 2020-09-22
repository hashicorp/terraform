package constraints

import (
	"bytes"
	"fmt"
	"strconv"
)

// Spec is an interface type that UnionSpec, IntersectionSpec, SelectionSpec,
// and VersionSpec all belong to.
//
// It's provided to allow generic code to be written that accepts and operates
// on all specs, but such code must still handle each type separately using
// e.g. a type switch. This is a closed type that will not have any new
// implementations added in future.
type Spec interface {
	isSpec()
}

// UnionSpec represents an "or" operation on nested version constraints.
//
// This is not directly representable in all of our supported constraint
// syntaxes.
type UnionSpec []IntersectionSpec

func (s UnionSpec) isSpec() {}

// IntersectionSpec represents an "and" operation on nested version constraints.
type IntersectionSpec []SelectionSpec

func (s IntersectionSpec) isSpec() {}

// SelectionSpec represents applying a single operator to a particular
// "boundary" version.
type SelectionSpec struct {
	Boundary VersionSpec
	Operator SelectionOp
}

func (s SelectionSpec) isSpec() {}

// VersionSpec represents the boundary within a SelectionSpec.
type VersionSpec struct {
	Major      NumConstraint
	Minor      NumConstraint
	Patch      NumConstraint
	Prerelease string
	Metadata   string
}

func (s VersionSpec) isSpec() {}

// IsExact returns bool if all of the version numbers in the receiver are
// fully-constrained. This is the same as s.ConstraintDepth() == ConstrainedPatch
func (s VersionSpec) IsExact() bool {
	return s.ConstraintDepth() == ConstrainedPatch
}

// ConstraintDepth returns the constraint depth of the receiver, which is
// the most specifc version number segment that is exactly constrained.
//
// The constraints must be consistent, which means that if a given segment
// is unconstrained then all of the deeper segments must also be unconstrained.
// If not, this method will panic. Version specs produced by the parsers in
// this package are guaranteed to be consistent.
func (s VersionSpec) ConstraintDepth() ConstraintDepth {
	if s == (VersionSpec{}) {
		// zero value is a degenerate case meaning completely unconstrained
		return Unconstrained
	}

	switch {
	case s.Major.Unconstrained:
		if !(s.Minor.Unconstrained && s.Patch.Unconstrained && s.Prerelease == "" && s.Metadata == "") {
			panic("inconsistent constraint depth")
		}
		return Unconstrained
	case s.Minor.Unconstrained:
		if !(s.Patch.Unconstrained && s.Prerelease == "" && s.Metadata == "") {
			panic("inconsistent constraint depth")
		}
		return ConstrainedMajor
	case s.Patch.Unconstrained:
		if s.Prerelease != "" || s.Metadata != "" {
			panic(fmt.Errorf("inconsistent constraint depth: wildcard major, minor and patch followed by prerelease %q and metadata %q", s.Prerelease, s.Metadata))
		}
		return ConstrainedMinor
	default:
		return ConstrainedPatch
	}
}

// ConstraintBounds returns two exact VersionSpecs that represent the upper
// and lower bounds of the possibly-inexact receiver. If the receiver
// is already exact then the two bounds are identical and have operator
// OpEqual. If they are different then the lower bound is OpGreaterThanOrEqual
// and the upper bound is OpLessThan.
//
// As a special case, if the version spec is entirely unconstrained the
// two bounds will be identical and the zero value of SelectionSpec. For
// consistency, this result is also returned if the receiver is already
// the zero value of VersionSpec, since a zero spec represents a lack of
// constraint.
//
// The constraints must be consistent as defined by ConstraintDepth, or this
// method will panic.
func (s VersionSpec) ConstraintBounds() (SelectionSpec, SelectionSpec) {
	switch s.ConstraintDepth() {
	case Unconstrained:
		return SelectionSpec{}, SelectionSpec{}
	case ConstrainedMajor:
		lowerBound := s.ConstrainToZero()
		lowerBound.Metadata = ""
		upperBound := lowerBound
		upperBound.Major.Num++
		upperBound.Minor.Num = 0
		upperBound.Patch.Num = 0
		upperBound.Prerelease = ""
		upperBound.Metadata = ""
		return SelectionSpec{
				Operator: OpGreaterThanOrEqual,
				Boundary: lowerBound,
			}, SelectionSpec{
				Operator: OpLessThan,
				Boundary: upperBound,
			}
	case ConstrainedMinor:
		lowerBound := s.ConstrainToZero()
		lowerBound.Metadata = ""
		upperBound := lowerBound
		upperBound.Minor.Num++
		upperBound.Patch.Num = 0
		upperBound.Metadata = ""
		return SelectionSpec{
				Operator: OpGreaterThanOrEqual,
				Boundary: lowerBound,
			}, SelectionSpec{
				Operator: OpLessThan,
				Boundary: upperBound,
			}
	default:
		eq := SelectionSpec{
			Operator: OpEqual,
			Boundary: s,
		}
		return eq, eq
	}
}

// ConstrainToZero returns a copy of the receiver with all of its
// unconstrained numeric segments constrained to zero.
func (s VersionSpec) ConstrainToZero() VersionSpec {
	switch s.ConstraintDepth() {
	case Unconstrained:
		s.Major = NumConstraint{Num: 0}
		s.Minor = NumConstraint{Num: 0}
		s.Patch = NumConstraint{Num: 0}
		s.Prerelease = ""
		s.Metadata = ""
	case ConstrainedMajor:
		s.Minor = NumConstraint{Num: 0}
		s.Patch = NumConstraint{Num: 0}
		s.Prerelease = ""
		s.Metadata = ""
	case ConstrainedMinor:
		s.Patch = NumConstraint{Num: 0}
		s.Prerelease = ""
		s.Metadata = ""
	}
	return s
}

// ConstrainToUpperBound returns a copy of the receiver with all of its
// unconstrained numeric segments constrained to zero and its last
// constrained segment increased by one.
//
// This operation is not meaningful for an entirely unconstrained VersionSpec,
// so will return the zero value of the type in that case.
func (s VersionSpec) ConstrainToUpperBound() VersionSpec {
	switch s.ConstraintDepth() {
	case Unconstrained:
		return VersionSpec{}
	case ConstrainedMajor:
		s.Major.Num++
		s.Minor = NumConstraint{Num: 0}
		s.Patch = NumConstraint{Num: 0}
		s.Prerelease = ""
		s.Metadata = ""
	case ConstrainedMinor:
		s.Minor.Num++
		s.Patch = NumConstraint{Num: 0}
		s.Prerelease = ""
		s.Metadata = ""
	}
	return s
}

func (s VersionSpec) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s.%s.%s", s.Major, s.Minor, s.Patch)
	if s.Prerelease != "" {
		fmt.Fprintf(&buf, "-%s", s.Prerelease)
	}
	if s.Metadata != "" {
		fmt.Fprintf(&buf, "+%s", s.Metadata)
	}
	return buf.String()
}

type SelectionOp rune

//go:generate stringer -type SelectionOp

const (
	OpUnconstrained               SelectionOp = 0
	OpGreaterThan                 SelectionOp = '>'
	OpLessThan                    SelectionOp = '<'
	OpGreaterThanOrEqual          SelectionOp = '≥'
	OpGreaterThanOrEqualPatchOnly SelectionOp = '~'
	OpGreaterThanOrEqualMinorOnly SelectionOp = '^'
	OpLessThanOrEqual             SelectionOp = '≤'
	OpEqual                       SelectionOp = '='
	OpNotEqual                    SelectionOp = '≠'
	OpMatch                       SelectionOp = '*'
)

type NumConstraint struct {
	Num           uint64
	Unconstrained bool
}

func (c NumConstraint) String() string {
	if c.Unconstrained {
		return "*"
	} else {
		return strconv.FormatUint(c.Num, 10)
	}
}

type ConstraintDepth int

//go:generate stringer -type ConstraintDepth

const (
	Unconstrained    ConstraintDepth = 0
	ConstrainedMajor ConstraintDepth = 1
	ConstrainedMinor ConstraintDepth = 2
	ConstrainedPatch ConstraintDepth = 3
)
