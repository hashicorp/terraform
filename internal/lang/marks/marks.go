package marks

import (
	"strings"

	"github.com/zclconf/go-cty/cty"
)

// valueMarks allow creating strictly typed values for use as cty.Value marks.
// The variable name for new values should be the title-cased format of the
// value to better match the GoString output for debugging.
type valueMark string

func (m valueMark) GoString() string {
	return "marks." + strings.Title(string(m))
}

// Has returns true if and only if the cty.Value has the given mark.
func Has(val cty.Value, mark valueMark) bool {
	return val.HasMark(mark)
}

// Contains returns true if the cty.Value or any any value within it contains
// the given mark.
func Contains(val cty.Value, mark valueMark) bool {
	ret := false
	cty.Walk(val, func(_ cty.Path, v cty.Value) (bool, error) {
		if v.HasMark(mark) {
			ret = true
			return false, nil
		}
		return true, nil
	})
	return ret
}

// Sensitive indicates that this value is marked as sensitive in the context of
// Terraform.
var Sensitive = valueMark("sensitive")

// Raw is used to indicate to the repl that the value should be written without
// any formatting.
var Raw = valueMark("raw")
