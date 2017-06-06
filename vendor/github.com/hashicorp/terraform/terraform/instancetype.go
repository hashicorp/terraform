package terraform

//go:generate stringer -type=InstanceType instancetype.go

// InstanceType is an enum of the various types of instances store in the State
type InstanceType int

const (
	TypeInvalid InstanceType = iota
	TypePrimary
	TypeTainted
	TypeDeposed
)
