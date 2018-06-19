package plans

type Action rune

const (
	NoOp    Action = 0
	Create  Action = '+'
	Read    Action = '←'
	Update  Action = '~'
	Replace Action = '±'
	Delete  Action = '-'
)

//go:generate stringer -type Action
