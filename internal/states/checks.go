package states

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

// Check represents the state of a check (preconditions, etc) as it
// was at the most recent update that included the object being checked.
//
// Terraform tracks these results primarily to allow external systems to
// consume this data (e.g. via the "terraform show -json" command) and use
// it to drive a check status UI or to send notifications of changes to check
// statuses.
//
// Applying planned changes and applying a refresh-only plan both create
// CheckState entries for any object that has custom checks. Planning also
// generates CheckState entries, although those will never be observed directly
// in a state snapshot except for snapshots embedded in plans.
type Check struct {
	// Object is the address of the object that the check belongs to.
	// For example, for a resource precondition or post condition this is the
	// address of the resource that the condition was configured against.
	Object addrs.Checkable

	// Status is the most recently determined status for the check. During
	// planning this can temporarily become CheckPending if the condition
	// expression depends on an unknown value, but in any stable state snapshot
	// it will always be either CheckPassed or CheckFailed.
	Status CheckStatus

	// ErrorMessage is the error message string configured by the author of
	// the check. This is populated even when the condition isn't failing in
	// order to allow potentially showing the message onscreen in an ongoing
	// status report, but the statement in the message will typically be true
	// only when Status is CheckFailed.
	ErrorMessage string
}

// CheckStatus is an enumeration representing the status of a check.
type CheckStatus rune

//go:generate go run golang.org/x/tools/cmd/stringer -type=CheckStatus checks.go
const (
	CheckPassed  CheckStatus = '✅'
	CheckFailed  CheckStatus = '🔥'
	CheckPending CheckStatus = '🤷'
)
