package stressgen

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

// ErrUnexpected is an implementation of error representing the situation where
// a result value didn't match expectations. It tracks both the actual result
// and the expected result, in case a caller wants to do use a special
// presentation for those values.
type ErrUnexpected struct {
	Message   string
	Got, Want interface{} // can be anything that go-spew can Sdump
}

func (err ErrUnexpected) Error() string {
	msg := err.Message
	if msg == "" {
		msg = "wrong result"
	}
	if gotV, ok := err.Got.(cty.Value); ok {
		if wantV, ok := err.Want.(cty.Value); ok {
			// If we're comparing two cty.Value then we'll use a more
			// specialized output structure that is optimized to make
			// them more readable than with spew, which tends to just
			// expose the ugly internals.
			return fmt.Sprintf("%s\ngot:  %swant: %s", msg, ctydebug.ValueString(gotV), ctydebug.ValueString(wantV))
		}
	}
	return fmt.Sprintf("%s\ngot: %swant: %s", msg, spew.Sdump(err.Got), spew.Sdump(err.Want))
}
