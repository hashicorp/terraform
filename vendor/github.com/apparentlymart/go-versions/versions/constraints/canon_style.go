package constraints

import (
	"fmt"
	"strings"
)

// Parse parses a constraint string using a syntax similar to that used by
// npm, Go "dep", Rust's "cargo", etc. Exact compatibility with any of these
// systems is not guaranteed, but instead we aim for familiarity in the choice
// of operators and their meanings. The syntax described here is considered the
// canonical syntax for this package, but a Ruby-style syntax is also offered
// via the function "ParseRubyStyle".
//
// A constraint string is a sequence of selection sets delimited by ||, with
// each selection set being a whitespace-delimited sequence of selections.
// Each selection is then the combination of a matching operator and a boundary
// version. The following is an example of a complex constraint string
// illustrating all of these features:
//
//     >=1.0.0 <2.0.0 || 1.0.0-beta1 || =2.0.2
//
// In practice constraint strings are usually simpler than this, but this
// complex example allows us to identify each of the parts by example:
//
//     Selection Sets:     ">=1.0.0 <2.0.0"
//                         "1.0.0-beta1"
//                         "=2.0.2"
//     Selections:         ">=1.0.0"
//                         "<2.0.0"
//                         "1.0.0-beta1"
//                         "=2.0.2"
//     Matching Operators: ">=", "<", "=" are explicit operators
//                         "1.0.0-beta1" has an implicit "=" operator
//     Boundary Versions:  "1.0.0", "2.0.0", "1.0.0-beta1", "2.0.2"
//
// A constraint string describes the members of a version set by adding exact
// versions or ranges of versions to that set. A version is in the set if
// any one of the selection sets match that version. A selection set matches
// a version if all of its selections match that version. A selection matches
// a version if the version has the indicated relationship with the given
// boundary version.
//
// In the above example, the first selection set matches all released versions
// whose major segment is 1, since both selections must apply. However, the
// remaining two selection sets describe two specific versions outside of that
// range that are also admitted, in addition to those in the indicated range.
//
// The available matching operators are:
//
//     <  Less than
//     <= Less than or equal
//     >  Greater than
//     >= Greater than or equal
//     =  Equal
//     !  Not equal
//     ~  Greater than with implied upper limit (described below)
//     ^  Greater than excluding new major releases (described below)
//
// If no operator is specified, the operator is implied to be "equal" for a
// full version specification, or a special additional "match" operator for
// a version containing wildcards as described below.
//
// The "~" matching operator is a shorthand for expressing both a lower and
// upper limit within a single expression. The effect of this operator depends
// on how many segments are specified in the boundary version: if only one
// segment is specified then new minor and patch versions are accepted, whereas
// if two or three segments are specified then only patch versions are accepted.
// For example:
//
//     ~1     is equivalent to >=1.0.0 <2.0.0
//     ~1.0   is equivalent to >=1.0.0 <1.1.0
//     ~1.2   is equivalent to >=1.2.0 <1.3.0
//     ~1.2.0 is equivalent to >=1.2.0 <1.3.0
//     ~1.2.3 is equivalent to >=1.2.3 <1.3.0
//
// The "^" matching operator is similar to "~" except that it always constrains
// only the major version number. It has an additional special behavior for
// when the major version number is zero: in that case, the minor release
// number is constrained, reflecting the common semver convention that initial
// development releases mark breaking changes by incrementing the minor version.
// For example:
//
//     ^1     is equivalent to >=1.0.0 <2.0.0
//     ^1.2   is equivalent to >=1.2.0 <2.0.0
//     ^1.2.3 is equivalent to >=1.2.3 <2.0.0
//     ^0.1.0 is equivalent to >=0.1.0 <0.2.0
//     ^0.1.2 is equivalent to >=0.1.2 <0.2.0
//
// The boundary version can contain wildcards for the major, minor or patch
// segments, which are specified using the markers "*", "x", or "X". When used
// in a selection with no explicit operator, these specify the implied "match"
// operator and define ranges with similar meaning to the "~" and "^" operators:
//
//     1.*    is equivalent to >=1.0.0 <2.0.0
//     1.*.*  is equivalent to >=1.0.0 <2.0.0
//     1.0.*  is equivalent to >=1.0.0 <1.1.0
//
// When wildcards are used, the first segment specified as a wildcard implies
// that all of the following segments are also wildcards. A version
// specification like "1.*.2" is invalid, because a wildcard minor version
// implies that the patch version must also be a wildcard.
//
// Wildcards have no special meaning when used with explicit operators, and so
// they are merely replaced with zeros in such cases.
//
// Explicit range syntax  using a hyphen creates inclusive upper and lower
// bounds:
//
//     1.0.0 - 2.0.0 is equivalent to >=1.0.0 <=2.0.0
//     1.2.3 - 2.3.4 is equivalent to >=1.2.3 <=2.3.4
//
// Requests of exact pre-release versions with the equals operator have
// no special meaning to the constraint parser, but are interpreted as explicit
// requests for those versions when interpreted by the MeetingConstraints
// function (and related functions) in the "versions" package, in the parent
// directory. Pre-release versions that are not explicitly requested are
// excluded from selection so that e.g. "^1.0.0" will not match a version
// "2.0.0-beta.1".
//
// The result is always a UnionSpec, whose members are IntersectionSpecs
// each describing one selection set. In the common case where a string
// contains only one selection, both the UnionSpec and the IntersectionSpec
// will have only one element and can thus be effectively ignored by the
// caller. (Union and intersection of single sets are both no-op.)
// A valid string must contain at least one selection; if an empty selection
// is to be considered as either "no versions" or "all versions" then this
// special case must be handled by the caller prior to calling this function.
//
// If there are syntax errors or ambiguities in the provided string then an
// error is returned. All errors returned by this function are suitable for
// display to English-speaking end-users, and avoid any Go-specific
// terminology.
func Parse(str string) (UnionSpec, error) {
	str = strings.TrimSpace(str)

	if str == "" {
		return nil, fmt.Errorf("empty specification")
	}

	// Most constraint strings contain only one selection, so we'll
	// allocate under that assumption and re-allocate if needed.
	uspec := make(UnionSpec, 0, 1)
	ispec := make(IntersectionSpec, 0, 1)

	remain := str
	for {
		var selection SelectionSpec
		var err error
		selection, remain, err = parseSelection(remain)
		if err != nil {
			return nil, err
		}

		remain = strings.TrimSpace(remain)

		if len(remain) > 0 && remain[0] == '-' {
			// Looks like user wants to make a range expression, so we'll
			// look for another selection.
			remain = strings.TrimSpace(remain[1:])
			if remain == "" {
				return nil, fmt.Errorf(`operator "-" must be followed by another version selection to specify the upper limit of the range`)
			}

			var lower, upper SelectionSpec
			lower = selection
			upper, remain, err = parseSelection(remain)
			remain = strings.TrimSpace(remain)
			if err != nil {
				return nil, err
			}

			if lower.Operator != OpUnconstrained {
				return nil, fmt.Errorf(`lower bound of range specified with "-" operator must be an exact version`)
			}
			if upper.Operator != OpUnconstrained {
				return nil, fmt.Errorf(`upper bound of range specified with "-" operator must be an exact version`)
			}

			lower.Operator = OpGreaterThanOrEqual
			lower.Boundary = lower.Boundary.ConstrainToZero()
			if upper.Boundary.IsExact() {
				upper.Operator = OpLessThanOrEqual
			} else {
				upper.Operator = OpLessThan
				upper.Boundary = upper.Boundary.ConstrainToUpperBound()
			}
			ispec = append(ispec, lower, upper)
		} else {
			if selection.Operator == OpUnconstrained {
				// Select a default operator based on whether the version
				// specification contains wildcards.
				if selection.Boundary.IsExact() {
					selection.Operator = OpEqual
				} else {
					selection.Operator = OpMatch
				}
			}
			if selection.Operator != OpMatch {
				switch selection.Operator {
				case OpMatch:
					// nothing to do
				case OpLessThanOrEqual:
					if !selection.Boundary.IsExact() {
						selection.Operator = OpLessThan
						selection.Boundary = selection.Boundary.ConstrainToUpperBound()
					}
				case OpGreaterThan:
					if !selection.Boundary.IsExact() {
						// If "greater than" has an imprecise boundary then we'll
						// turn it into a "greater than or equal to" and use the
						// upper bound of the boundary, so e.g.:
						// >1.*.* means >=2.0.0, because that's greater than
						// everything matched by 1.*.*.
						selection.Operator = OpGreaterThanOrEqual
						selection.Boundary = selection.Boundary.ConstrainToUpperBound()
					}
				default:
					selection.Boundary = selection.Boundary.ConstrainToZero()
				}
			}
			ispec = append(ispec, selection)
		}

		if len(remain) == 0 {
			// All done!
			break
		}

		if remain[0] == ',' {
			return nil, fmt.Errorf(`commas are not needed to separate version selections; separate with spaces instead`)
		}

		if remain[0] == '|' {
			if !strings.HasPrefix(remain, "||") {
				// User was probably trying for "||", so we'll produce a specialized error
				return nil, fmt.Errorf(`single "|" is not a valid operator; did you mean "||" to specify an alternative?`)
			}
			remain = strings.TrimSpace(remain[2:])
			if remain == "" {
				return nil, fmt.Errorf(`operator "||" must be followed by another version selection`)
			}

			// Begin a new IntersectionSpec, added to our single UnionSpec
			uspec = append(uspec, ispec)
			ispec = make(IntersectionSpec, 0, 1)
		}
	}

	uspec = append(uspec, ispec)

	return uspec, nil
}

// parseSelection parses one canon-style selection from the prefix of the
// given string, returning the result along with the remaining unconsumed
// string for the caller to use for further processing.
func parseSelection(str string) (SelectionSpec, string, error) {
	raw, remain := scanConstraint(str)
	var spec SelectionSpec

	if len(str) == len(remain) {
		if len(remain) > 0 && remain[0] == 'v' {
			// User seems to be trying to use a "v" prefix, like "v1.0.0"
			return spec, remain, fmt.Errorf(`a "v" prefix should not be used when specifying versions`)
		}

		// If we made no progress at all then the selection must be entirely invalid.
		return spec, remain, fmt.Errorf("the sequence %q is not valid", remain)
	}

	switch raw.op {
	case "":
		// We'll deal with this situation in the caller
		spec.Operator = OpUnconstrained
	case "=":
		spec.Operator = OpEqual
	case "!":
		spec.Operator = OpNotEqual
	case ">":
		spec.Operator = OpGreaterThan
	case ">=":
		spec.Operator = OpGreaterThanOrEqual
	case "<":
		spec.Operator = OpLessThan
	case "<=":
		spec.Operator = OpLessThanOrEqual
	case "~":
		if raw.numCt > 1 {
			spec.Operator = OpGreaterThanOrEqualPatchOnly
		} else {
			spec.Operator = OpGreaterThanOrEqualMinorOnly
		}
	case "^":
		if len(raw.nums[0]) > 0 && raw.nums[0][0] == '0' {
			// Special case for major version 0, which is initial development:
			// we treat the minor number as if it's the major number.
			spec.Operator = OpGreaterThanOrEqualPatchOnly
		} else {
			spec.Operator = OpGreaterThanOrEqualMinorOnly
		}
	case "=<":
		return spec, remain, fmt.Errorf("invalid constraint operator %q; did you mean \"<=\"?", raw.op)
	case "=>":
		return spec, remain, fmt.Errorf("invalid constraint operator %q; did you mean \">=\"?", raw.op)
	default:
		return spec, remain, fmt.Errorf("invalid constraint operator %q", raw.op)
	}

	if raw.sep != "" {
		return spec, remain, fmt.Errorf("no spaces allowed after operator %q", raw.op)
	}

	if raw.numCt > 3 {
		return spec, remain, fmt.Errorf("too many numbered portions; only three are allowed (major, minor, patch)")
	}

	// Unspecified portions are either zero or wildcard depending on whether
	// any explicit wildcards are present.
	seenWild := false
	for i, s := range raw.nums {
		switch {
		case isWildcardNum(s):
			seenWild = true
		case i >= raw.numCt:
			if seenWild {
				raw.nums[i] = "*"
			} else {
				raw.nums[i] = "0"
			}
		default:
			// If we find a non-wildcard after we've already seen a wildcard
			// then this specification is inconsistent, which is an error.
			if seenWild {
				return spec, remain, fmt.Errorf("can't use exact %s segment after a previous segment was wildcard", rawNumNames[i])
			}
		}
	}

	if seenWild {
		if raw.pre != "" {
			return spec, remain, fmt.Errorf(`can't use prerelease segment (introduced by "-") in a version with wildcards`)
		}
		if raw.meta != "" {
			return spec, remain, fmt.Errorf(`can't use build metadata segment (introduced by "+") in a version with wildcards`)
		}
	}

	spec.Boundary = raw.VersionSpec()

	return spec, remain, nil
}
