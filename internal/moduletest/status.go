package moduletest

import "github.com/mitchellh/colorstring"

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

// ColorizedText returns textual representations of the status, but colored for
// visual appeal.
func (status Status) ColorizedText(color *colorstring.Colorize) string {
	switch status {
	case Error, Fail:
		return color.Color("[red]fail[reset]")
	case Pass:
		return color.Color("[green]pass[reset]")
	case Skip:
		return color.Color("[light_gray]skip[reset]")
	case Pending:
		return color.Color("[light_gray]pending[reset]")
	default:
		panic("unrecognized status: " + status.String())
	}
}
