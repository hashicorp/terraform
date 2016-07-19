package vsphere

import "strings"

type stringer interface {
	String() string
}

// JoinStringer joins stringer elements like strings.Join
func JoinStringer(values []stringer, sep string) string {
	var data = make([]string, len(values))
	for i, v := range values {
		data[i] = v.String()
	}
	return strings.Join(data, sep)
}
