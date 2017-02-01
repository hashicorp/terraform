package local

//go:generate stringer -type=countHookAction hook_count_action.go

type countHookAction byte

const (
	countHookActionAdd countHookAction = iota
	countHookActionChange
	countHookActionRemove
)
