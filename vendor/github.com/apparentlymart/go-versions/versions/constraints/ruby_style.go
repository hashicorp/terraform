package constraints

import (
	"fmt"
	"strings"
)

// ParseRubyStyle parses a single selection constraint using a syntax similar
// to that used by rubygems and other Ruby tools.
//
// Exact compatibility with rubygems is not guaranteed; "ruby-style" here
// just means that users familiar with rubygems should find familiar the choice
// of operators and their meanings.
//
// ParseRubyStyle parses only a single specification, mimicking the usual
// rubygems approach of providing each selection as a separate string.
// The result can be combined with other results to create an IntersectionSpec
// that describes the effect of multiple such constraints.
func ParseRubyStyle(str string) (SelectionSpec, error) {
	if strings.TrimSpace(str) == "" {
		return SelectionSpec{}, fmt.Errorf("empty specification")
	}
	spec, remain, err := parseRubyStyle(str)
	if err != nil {
		return spec, err
	}
	if remain != "" {
		remain = strings.TrimSpace(remain)
		switch {
		case remain == "":
			return spec, fmt.Errorf("extraneous spaces at end of specification")
		case strings.HasPrefix(remain, "v"):
			// User seems to be trying to use a "v" prefix, like "v1.0.0"
			return spec, fmt.Errorf(`a "v" prefix should not be used`)
		case strings.HasPrefix(remain, "||") || strings.HasPrefix(remain, ","):
			// User seems to be trying to specify multiple constraints
			return spec, fmt.Errorf(`only one constraint may be specified`)
		case strings.HasPrefix(remain, "-"):
			// User seems to be trying to use npm-style range constraints
			return spec, fmt.Errorf(`range constraints are not supported`)
		default:
			return spec, fmt.Errorf("invalid characters %q", remain)
		}
	}

	return spec, nil
}

// ParseRubyStyleAll is a helper wrapper around ParseRubyStyle that accepts
// multiple selection strings and combines them together into a single
// IntersectionSpec.
func ParseRubyStyleAll(strs ...string) (IntersectionSpec, error) {
	spec := make(IntersectionSpec, 0, len(strs))
	for _, str := range strs {
		subSpec, err := ParseRubyStyle(str)
		if err != nil {
			return nil, fmt.Errorf("invalid specification %q: %s", str, err)
		}
		spec = append(spec, subSpec)
	}
	return spec, nil
}

// ParseRubyStyleMulti is similar to ParseRubyStyle, but rather than parsing
// only a single selection specification it instead expects one or more
// comma-separated specifications, returning the result as an
// IntersectionSpec.
func ParseRubyStyleMulti(str string) (IntersectionSpec, error) {
	var spec IntersectionSpec
	remain := strings.TrimSpace(str)
	for remain != "" {
		if strings.TrimSpace(remain) == "" {
			break
		}

		var subSpec SelectionSpec
		var err error
		var newRemain string
		subSpec, newRemain, err = parseRubyStyle(remain)
		consumed := remain[:len(remain)-len(newRemain)]
		if err != nil {
			return nil, fmt.Errorf("invalid specification %q: %s", consumed, err)
		}
		remain = strings.TrimSpace(newRemain)

		if remain != "" {
			if !strings.HasPrefix(remain, ",") {
				return nil, fmt.Errorf("missing comma after %q", consumed)
			}
			// Eat the separator comma
			remain = strings.TrimSpace(remain[1:])
		}

		spec = append(spec, subSpec)
	}

	return spec, nil
}

// parseRubyStyle parses a ruby-style constraint from the prefix of the given
// string and returns the remaining unconsumed string for the caller to use
// for further processing.
func parseRubyStyle(str string) (SelectionSpec, string, error) {
	raw, remain := scanConstraint(str)
	var spec SelectionSpec

	switch raw.op {
	case "=", "":
		spec.Operator = OpEqual
	case "!=":
		spec.Operator = OpNotEqual
	case ">":
		spec.Operator = OpGreaterThan
	case ">=":
		spec.Operator = OpGreaterThanOrEqual
	case "<":
		spec.Operator = OpLessThan
	case "<=":
		spec.Operator = OpLessThanOrEqual
	case "~>":
		// Ruby-style pessimistic can be either a minor-only or patch-only
		// constraint, depending on how many digits were given.
		switch raw.numCt {
		case 3:
			spec.Operator = OpGreaterThanOrEqualPatchOnly
		default:
			spec.Operator = OpGreaterThanOrEqualMinorOnly
		}
	case "=<":
		return spec, remain, fmt.Errorf("invalid constraint operator %q; did you mean \"<=\"?", raw.op)
	case "=>":
		return spec, remain, fmt.Errorf("invalid constraint operator %q; did you mean \">=\"?", raw.op)
	default:
		return spec, remain, fmt.Errorf("invalid constraint operator %q", raw.op)
	}

	switch raw.sep {
	case "":
		// No separator is always okay. Although all of the examples in the
		// rubygems docs show a space separator, the parser doesn't actually
		// require it.
	case " ":
		if raw.op == "" {
			return spec, remain, fmt.Errorf("extraneous spaces at start of specification")
		}
	default:
		if raw.op == "" {
			return spec, remain, fmt.Errorf("extraneous spaces at start of specification")
		} else {
			return spec, remain, fmt.Errorf("only one space is expected after the operator %q", raw.op)
		}
	}

	if raw.numCt > 3 {
		return spec, remain, fmt.Errorf("too many numbered portions; only three are allowed (major, minor, patch)")
	}

	// Ruby-style doesn't use explicit wildcards
	for i, s := range raw.nums {
		switch {
		case isWildcardNum(s):
			// Can't use wildcards in an exact specification
			return spec, remain, fmt.Errorf("can't use wildcard for %s number; omit segments that should be unconstrained", rawNumNames[i])
		}
	}

	if raw.pre != "" || raw.meta != "" {
		// If either the prerelease or meta portions are set then any unconstrained
		// segments are implied to be zero in order to guarantee constraint
		// consistency.
		for i, s := range raw.nums {
			if s == "" {
				raw.nums[i] = "0"
			}
		}
	}

	spec.Boundary = raw.VersionSpec()

	return spec, remain, nil
}
