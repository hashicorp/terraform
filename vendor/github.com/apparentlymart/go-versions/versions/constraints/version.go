package constraints

import (
	"fmt"
	"strings"
)

// ParseExactVersion parses a string that must contain the specification of a
// single, exact version, and then returns it as a VersionSpec.
//
// This is primarily here to allow versions.ParseVersion to re-use the
// constraint grammar, and isn't very useful for direct use from calling
// applications.
func ParseExactVersion(vs string) (VersionSpec, error) {
	spec := VersionSpec{}

	if strings.TrimSpace(vs) == "" {
		return spec, fmt.Errorf("empty specification")
	}

	raw, remain := scanConstraint(vs)

	switch strings.TrimSpace(raw.op) {
	case ">", ">=", "<", "<=", "!", "!=", "~>", "^", "~":
		// If it looks like the user was trying to write a constraint string
		// then we'll help them out with a more specialized error.
		return spec, fmt.Errorf("can't use constraint operator %q; an exact version is required", raw.op)
	case "":
		// Empty operator is okay as long as we don't also have separator spaces.
		// (Caller can trim off spaces beforehand if they want to tolerate this.)
		if raw.sep != "" {
			return spec, fmt.Errorf("extraneous spaces at start of specification")
		}
	default:
		return spec, fmt.Errorf("invalid sequence %q at start of specification", raw.op)
	}

	if remain != "" {
		remain = strings.TrimSpace(remain)
		switch {
		case remain == "":
			return spec, fmt.Errorf("extraneous spaces at end of specification")
		case strings.HasPrefix(vs, "v"):
			// User seems to be trying to use a "v" prefix, like "v1.0.0"
			return spec, fmt.Errorf(`a "v" prefix should not be used`)
		case strings.HasPrefix(remain, ",") || strings.HasPrefix(remain, "|"):
			// User seems to be trying to list/combine multiple versions
			return spec, fmt.Errorf("can't specify multiple versions; a single exact version is required")
		case strings.HasPrefix(remain, "-"):
			// User seems to be trying to use the npm-style range operator
			return spec, fmt.Errorf("can't specify version range; a single exact version is required")
		case strings.HasPrefix(strings.TrimSpace(vs), remain):
			// Whole string is invalid, then.
			return spec, fmt.Errorf("invalid specification; required format is three positive integers separated by periods")
		default:
			return spec, fmt.Errorf("invalid characters %q", remain)
		}
	}

	if raw.numCt > 3 {
		return spec, fmt.Errorf("too many numbered portions; only three are allowed (major, minor, patch)")
	}

	for i := raw.numCt; i < len(raw.nums); i++ {
		raw.nums[i] = "0"
	}

	for i, s := range raw.nums {
		switch {
		case isWildcardNum(s):
			// Can't use wildcards in an exact specification
			return spec, fmt.Errorf("can't use wildcard for %s number; an exact version is required", rawNumNames[i])
		}
	}

	// Since we eliminated all of the unconstrained cases above, either by normalizing
	// or returning an error, we are guaranteed to get constrained numbers here.
	spec = raw.VersionSpec()

	return spec, nil
}
