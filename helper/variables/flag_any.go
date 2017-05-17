package variables

import (
	"strings"
)

// FlagAny is a flag.Value for parsing user variables in the format of
// 'key=value' OR a file path. 'key=value' is assumed if '=' is in the value.
// You cannot use a file path that contains an '='.
type FlagAny map[string]interface{}

func (v *FlagAny) String() string {
	return ""
}

func (v *FlagAny) Set(raw string) error {
	idx := strings.Index(raw, "=")
	if idx >= 0 {
		flag := (*Flag)(v)
		return flag.Set(raw)
	}

	flag := (*FlagFile)(v)
	return flag.Set(raw)
}
