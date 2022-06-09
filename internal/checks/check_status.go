package checks

// CheckStatus is an enumeration representing the status of a check.
type CheckStatus rune

//go:generate go run golang.org/x/tools/cmd/stringer -type=CheckStatus checks.go
const (
	// CheckPassed means that a check condition was evaluated and returned true.
	CheckPassed CheckStatus = '✅'

	// CheckFailed means that a check condition was evaluated and returned false.
	CheckFailed CheckStatus = '🔥'

	// CheckPending represents one of the following situations:
	//   - a check's condition hasn't been evaluated yet, because a graph walk
	//     is in progress.
	//   - a check's condition was evaluated but it returned an unknown value
	//     instead of a definitive result.
	//   - an error blocked either the evaluation of the check's condition or
	//     of something upstream of it.
	//
	// In all of those cases, the intent is that either continuing or retrying
	// evaluation (possibly after fixing a configuration error) should
	// eventually drive the result to either CheckPassed or CheckFailed.
	CheckPending CheckStatus = '🤷'
)
