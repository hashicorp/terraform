package local

//go:generate go run golang.org/x/tools/cmd/stringer -type=countHookAction hook_count_action.go

type countHookAction byte

const (
	countHookActionAdd countHookAction = iota
	countHookActionChange
	countHookActionRemove
)
