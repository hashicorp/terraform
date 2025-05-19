// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package providerreqs contains types we use to talk about provider
// requirements.
//
// This is separated from the parent directory package getproviders because
// lots of Terraform packages need to talk about provider requirements but
// very few actually need to perform provider plugin installation, and so
// this separate package avoids the need for every package that talks about
// provider requirements to also indirectly depend on all of the external
// modules used for provider installation.
package providerreqs

import (
	"fmt"
	"sort"
	"strings"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/apparentlymart/go-versions/versions/constraints"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Version represents a particular single version of a provider.
type Version = versions.Version

// UnspecifiedVersion is the zero value of Version, representing the absense
// of a version number.
var UnspecifiedVersion Version = versions.Unspecified

// VersionList represents a list of versions. It is a []Version with some
// extra methods for convenient filtering.
type VersionList = versions.List

// VersionSet represents a set of versions, usually describing the acceptable
// versions that can be selected under a particular version constraint provided
// by the end-user.
type VersionSet = versions.Set

// VersionConstraints represents a set of version constraints, which can
// define the membership of a VersionSet by exclusion.
type VersionConstraints = constraints.IntersectionSpec

// Warnings represents a list of warnings returned by a Registry source.
type Warnings = []string

// Requirements gathers together requirements for many different providers
// into a single data structure, as a convenient way to represent the full
// set of requirements for a particular configuration or state or both.
//
// If an entry in a Requirements has a zero-length VersionConstraints then
// that indicates that the provider is required but that any version is
// acceptable. That's different than a provider being absent from the map
// altogether, which means that it is not required at all.
type Requirements map[addrs.Provider]VersionConstraints

// Merge takes the requirements in the receiever and the requirements in the
// other given value and produces a new set of requirements that combines
// all of the requirements of both.
//
// The resulting requirements will permit only selections that both of the
// source requirements would've allowed.
func (r Requirements) Merge(other Requirements) Requirements {
	ret := make(Requirements)
	for addr, constraints := range r {
		ret[addr] = constraints
	}
	for addr, constraints := range other {
		ret[addr] = append(ret[addr], constraints...)
	}
	return ret
}

// Selections gathers together version selections for many different providers.
//
// This is the result of provider installation: a specific version selected
// for each provider given in the requested Requirements, selected based on
// the given version constraints.
type Selections map[addrs.Provider]Version

// ParseVersion parses a "semver"-style version string into a Version value,
// which is the version syntax we use for provider versions.
func ParseVersion(str string) (Version, error) {
	return versions.ParseVersion(str)
}

// MustParseVersion is a variant of ParseVersion that panics if it encounters
// an error while parsing.
func MustParseVersion(str string) Version {
	ret, err := ParseVersion(str)
	if err != nil {
		panic(err)
	}
	return ret
}

// ParseVersionConstraints parses a "Ruby-like" version constraint string
// into a VersionConstraints value.
func ParseVersionConstraints(str string) (VersionConstraints, error) {
	return constraints.ParseRubyStyleMulti(str)
}

// MustParseVersionConstraints is a variant of ParseVersionConstraints that
// panics if it encounters an error while parsing.
func MustParseVersionConstraints(str string) VersionConstraints {
	ret, err := ParseVersionConstraints(str)
	if err != nil {
		panic(err)
	}
	return ret
}

// MeetingConstraints returns a version set that contains all of the versions
// that meet the given constraints, specified using the Spec type from the
// constraints package.
func MeetingConstraints(vc VersionConstraints) VersionSet {
	return versions.MeetingConstraints(vc)
}

// VersionConstraintsString returns a canonical string representation of
// a VersionConstraints value.
func VersionConstraintsString(spec VersionConstraints) string {
	// (we have our own function for this because the upstream versions
	// library prefers to use npm/cargo-style constraint syntax, but
	// Terraform prefers Ruby-like. Maybe we can upstream a "RubyLikeString")
	// function to do this later, but having this in here avoids blocking on
	// that and this is the sort of thing that is unlikely to need ongoing
	// maintenance because the version constraint syntax is unlikely to change.)
	//
	// ParseVersionConstraints allows some variations for convenience, but the
	// return value from this function serves as the normalized form of a
	// particular version constraint, which is the form we require in dependency
	// lock files. Therefore the canonical forms produced here are a compatibility
	// constraint for the dependency lock file parser.

	if len(spec) == 0 {
		return ""
	}

	// VersionConstraints values are typically assembled by combining together
	// the version constraints from many separate declarations throughout
	// a configuration, across many modules. As a consequence, they typically
	// contain duplicates and the terms inside are in no particular order.
	// For our canonical representation we'll both deduplicate the items
	// and sort them into a consistent order.
	sels := make(map[constraints.SelectionSpec]struct{})
	for _, sel := range spec {
		// The parser allows writing abbreviated version (such as 2) which
		// end up being represented in memory with trailing unconstrained parts
		// (for example 2.*.*). For the purpose of serialization with Ruby
		// style syntax, these unconstrained parts can all be represented as 0
		// with no loss of meaning, so we make that conversion here. Doing so
		// allows us to deduplicate equivalent constraints, such as >= 2.0 and
		// >= 2.0.0.
		normalizedSel := constraints.SelectionSpec{
			Operator: sel.Operator,
			Boundary: sel.Boundary.ConstrainToZero(),
		}
		sels[normalizedSel] = struct{}{}
	}
	selsOrder := make([]constraints.SelectionSpec, 0, len(sels))
	for sel := range sels {
		selsOrder = append(selsOrder, sel)
	}
	sort.Slice(selsOrder, func(i, j int) bool {
		is, js := selsOrder[i], selsOrder[j]
		boundaryCmp := versionSelectionBoundaryCompare(is.Boundary, js.Boundary)
		if boundaryCmp == 0 {
			// The operator is the decider, then.
			return versionSelectionOperatorLess(is.Operator, js.Operator)
		}
		return boundaryCmp < 0
	})

	var b strings.Builder
	for i, sel := range selsOrder {
		if i > 0 {
			b.WriteString(", ")
		}
		switch sel.Operator {
		case constraints.OpGreaterThan:
			b.WriteString("> ")
		case constraints.OpLessThan:
			b.WriteString("< ")
		case constraints.OpGreaterThanOrEqual:
			b.WriteString(">= ")
		case constraints.OpGreaterThanOrEqualPatchOnly, constraints.OpGreaterThanOrEqualMinorOnly:
			// These two differ in how the version is written, not in the symbol.
			b.WriteString("~> ")
		case constraints.OpLessThanOrEqual:
			b.WriteString("<= ")
		case constraints.OpEqual:
			b.WriteString("")
		case constraints.OpNotEqual:
			b.WriteString("!= ")
		default:
			// The above covers all of the operators we support during
			// parsing, so we should not get here.
			b.WriteString("??? ")
		}

		// We use a different constraint operator to distinguish between the
		// two types of pessimistic constraint: minor-only and patch-only. For
		// minor-only constraints, we always want to display only the major and
		// minor version components, so we special-case that operator below.
		//
		// One final edge case is a minor-only constraint specified with only
		// the major version, such as ~> 2. We treat this the same as ~> 2.0,
		// because a major-only pessimistic constraint does not exist: it is
		// logically identical to >= 2.0.0.
		if sel.Operator == constraints.OpGreaterThanOrEqualMinorOnly {
			// The minor-pessimistic syntax uses only two version components.
			fmt.Fprintf(&b, "%s.%s", sel.Boundary.Major, sel.Boundary.Minor)
		} else {
			fmt.Fprintf(&b, "%s.%s.%s", sel.Boundary.Major, sel.Boundary.Minor, sel.Boundary.Patch)
		}
		if sel.Boundary.Prerelease != "" {
			b.WriteString("-" + sel.Boundary.Prerelease)
		}
		if sel.Boundary.Metadata != "" {
			b.WriteString("+" + sel.Boundary.Metadata)
		}
	}
	return b.String()
}

// Our sort for selection operators is somewhat arbitrary and mainly motivated
// by consistency rather than meaning, but this ordering does at least try
// to make it so "simple" constraint sets will appear how a human might
// typically write them, with the lower bounds first and the upper bounds
// last. Weird mixtures of different sorts of constraints will likely seem
// less intuitive, but they'd be unintuitive no matter the ordering.
var versionSelectionsBoundaryPriority = map[constraints.SelectionOp]int{
	// We skip zero here so that if we end up seeing an invalid
	// operator (which the string function would render as "???")
	// then it will have index zero and thus appear first.
	constraints.OpGreaterThan:                 1,
	constraints.OpGreaterThanOrEqual:          2,
	constraints.OpEqual:                       3,
	constraints.OpGreaterThanOrEqualPatchOnly: 4,
	constraints.OpGreaterThanOrEqualMinorOnly: 5,
	constraints.OpLessThanOrEqual:             6,
	constraints.OpLessThan:                    7,
	constraints.OpNotEqual:                    8,
}

func versionSelectionOperatorLess(i, j constraints.SelectionOp) bool {
	iPrio := versionSelectionsBoundaryPriority[i]
	jPrio := versionSelectionsBoundaryPriority[j]
	return iPrio < jPrio
}

func versionSelectionBoundaryCompare(i, j constraints.VersionSpec) int {
	// In the Ruby-style constraint syntax, unconstrained parts appear
	// only for omitted portions of a version string, like writing
	// "2" instead of "2.0.0". For sorting purposes we'll just
	// consider those as zero, which also matches how we serialize them
	// to strings.
	i, j = i.ConstrainToZero(), j.ConstrainToZero()

	// Once we've removed any unconstrained parts, we can safely
	// convert to our main Version type so we can use its ordering.
	iv := Version{
		Major:      i.Major.Num,
		Minor:      i.Minor.Num,
		Patch:      i.Patch.Num,
		Prerelease: versions.VersionExtra(i.Prerelease),
		Metadata:   versions.VersionExtra(i.Metadata),
	}
	jv := Version{
		Major:      j.Major.Num,
		Minor:      j.Minor.Num,
		Patch:      j.Patch.Num,
		Prerelease: versions.VersionExtra(j.Prerelease),
		Metadata:   versions.VersionExtra(j.Metadata),
	}
	if iv.Same(jv) {
		// Although build metadata doesn't normally weigh in to
		// precedence choices, we'll use it for our visual
		// ordering just because we need to pick _some_ order.
		switch {
		case iv.Metadata.Raw() == jv.Metadata.Raw():
			return 0
		case iv.Metadata.LessThan(jv.Metadata):
			return -1
		default:
			return 1 // greater, by elimination
		}
	}
	switch {
	case iv.LessThan(jv):
		return -1
	default:
		return 1 // greater, by elimination
	}
}
