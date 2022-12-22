package providermocks

import "fmt"

type requestType rune

const (
	readRequest  requestType = 'R'
	planRequest  requestType = 'P'
	applyRequest requestType = 'A'
)

//go:generate go run golang.org/x/tools/cmd/stringer -type requestType

func (rt requestType) BlockTypeName() string {
	switch rt {
	case readRequest:
		return "read"
	case planRequest:
		return "plan"
	case applyRequest:
		return "apply"
	default:
		panic(fmt.Sprintf("Invalid request type %s", rt.String()))
	}
}
