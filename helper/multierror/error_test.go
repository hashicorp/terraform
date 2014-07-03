package multierror

import (
	"errors"
	"testing"
)

func TestError_Impl(t *testing.T) {
	var raw interface{}
	raw = &Error{}
	if _, ok := raw.(error); !ok {
		t.Fatal("Error must implement error")
	}
}

func TestErrorError(t *testing.T) {
	expected := `2 error(s) occurred:

* foo
* bar`

	errors := []error{
		errors.New("foo"),
		errors.New("bar"),
	}

	multi := &Error{errors}
	if multi.Error() != expected {
		t.Fatalf("bad: %s", multi.Error())
	}
}

func TestErrorAppend_Error(t *testing.T) {
	original := &Error{
		Errors: []error{errors.New("foo")},
	}

	result := ErrorAppend(original, errors.New("bar"))
	if len(result.Errors) != 2 {
		t.Fatalf("wrong len: %d", len(result.Errors))
	}

	original = &Error{}
	result = ErrorAppend(original, errors.New("bar"))
	if len(result.Errors) != 1 {
		t.Fatalf("wrong len: %d", len(result.Errors))
	}
}

func TestErrorAppend_NonError(t *testing.T) {
	original := errors.New("foo")
	result := ErrorAppend(original, errors.New("bar"))
	if len(result.Errors) != 2 {
		t.Fatalf("wrong len: %d", len(result.Errors))
	}
}
