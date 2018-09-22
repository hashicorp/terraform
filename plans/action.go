package plans

type Action rune

const (
	NoOp             Action = 0
	Create           Action = '+'
	Read             Action = '←'
	Update           Action = '~'
	DeleteThenCreate Action = '∓'
	CreateThenDelete Action = '±'
	Delete           Action = '-'
)

//go:generate stringer -type Action

// IsReplace returns true if the action is one of the two actions that
// represents replacing an existing object with a new object:
// DeleteThenCreate or CreateThenDelete.
func (a Action) IsReplace() bool {
	return a == DeleteThenCreate || a == CreateThenDelete
}
