package marks

import (
	"strings"
)

// valueMarks allow creating strictly typed values for use as cty.Value marks.
// The variable name for new values should be the title-cased format of the
// value to better match the GoString output for debugging.
type valueMark string

func (m valueMark) GoString() string {
	return "marks." + strings.Title(string(m))
}

// Sensitive indicates that this value is marked as sensitive in the context of
// Terraform.
var Sensitive = valueMark("sensitive")
