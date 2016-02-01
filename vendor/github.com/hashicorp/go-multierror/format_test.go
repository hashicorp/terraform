package multierror

import (
	"errors"
	"testing"
)

func TestListFormatFunc(t *testing.T) {
	expected := `2 error(s) occurred:

* foo
* bar`

	errors := []error{
		errors.New("foo"),
		errors.New("bar"),
	}

	actual := ListFormatFunc(errors)
	if actual != expected {
		t.Fatalf("bad: %#v", actual)
	}
}
