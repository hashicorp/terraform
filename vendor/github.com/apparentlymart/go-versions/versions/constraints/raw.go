package constraints

import (
	"strconv"
)

//go:generate ragel -G1 -Z raw_scan.rl
//go:generate gofmt -w raw_scan.go

// rawConstraint is a tokenization of a constraint string, used internally
// as the first layer of parsing.
type rawConstraint struct {
	op    string
	sep   string
	nums  [3]string
	numCt int
	pre   string
	meta  string
}

// VersionSpec turns the receiver into a VersionSpec in a reasonable
// default way. This method assumes that the raw constraint was already
// validated, and will panic or produce undefined results if it contains
// anything invalid.
//
// In particular, numbers are automatically marked as unconstrained if they
// are omitted or set to wildcards, so the caller must apply any additional
// validation rules on the usage of unconstrained numbers before calling.
func (raw rawConstraint) VersionSpec() VersionSpec {
	return VersionSpec{
		Major:      parseRawNumConstraint(raw.nums[0]),
		Minor:      parseRawNumConstraint(raw.nums[1]),
		Patch:      parseRawNumConstraint(raw.nums[2]),
		Prerelease: raw.pre,
		Metadata:   raw.meta,
	}
}

var rawNumNames = [...]string{"major", "minor", "patch"}

func isWildcardNum(s string) bool {
	switch s {
	case "*", "x", "X":
		return true
	default:
		return false
	}
}

// parseRawNum parses a raw number string which the caller has already
// determined is non-empty and non-wildcard. If the string is not numeric
// then this function will panic.
func parseRawNum(s string) uint64 {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return v
}

// parseRawNumConstraint parses a raw number into a NumConstraint, setting it
// to unconstrained if the value is empty or a wildcard.
func parseRawNumConstraint(s string) NumConstraint {
	switch {
	case s == "" || isWildcardNum(s):
		return NumConstraint{
			Unconstrained: true,
		}
	default:
		return NumConstraint{
			Num: parseRawNum(s),
		}
	}
}
