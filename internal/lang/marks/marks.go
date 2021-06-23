package marks

import (
	"strings"
)

type valueMark string

func (m valueMark) GoString() string {
	return "marks." + strings.Title(string(m))
}

var Sensitive = valueMark("sensitive")
