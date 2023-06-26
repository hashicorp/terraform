package moduletest

// Status represents the status of a test case, and is defined as an iota within
// this file.
//
// The order of the definitions matter as different statuses do naturally take
// precedence over others. A test suite that has a mix of pass and fail statuses
// has failed overall and therefore the fail status is of higher precedence than
// the pass status.
//
// See the Status.Merge function for this requirement being used in action.
//
//go:generate go run golang.org/x/tools/cmd/stringer -type=Status status.go
type Status int

const (
	Pending Status = iota
	Skip
	Pass
	Fail
	Error
)

// Merge compares two statuses and returns a status that best represents the two
// together.
//
// This should be used to collate the overall status of a test file or test
// suite from the collection of test runs that have been executed.
//
// Essentially, if a test suite has a bunch of failures and passes the overall
// status would be failure. If a test suite has all passes, then the test suite
// would be pass overall.
//
// The implementation basically always returns the highest of the two, which
// means the order the statuses are defined within the iota matters.
func (status Status) Merge(next Status) Status {
	if next > status {
		return next
	}
	return status
}
