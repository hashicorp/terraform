package multierror

import (
	"errors"
	"testing"
)

func TestAppend_Error(t *testing.T) {
	original := &Error{
		Errors: []error{errors.New("foo")},
	}

	result := Append(original, errors.New("bar"))
	if len(result.Errors) != 2 {
		t.Fatalf("wrong len: %d", len(result.Errors))
	}

	original = &Error{}
	result = Append(original, errors.New("bar"))
	if len(result.Errors) != 1 {
		t.Fatalf("wrong len: %d", len(result.Errors))
	}

	// Test when a typed nil is passed
	var e *Error
	result = Append(e, errors.New("baz"))
	if len(result.Errors) != 1 {
		t.Fatalf("wrong len: %d", len(result.Errors))
	}

	// Test flattening
	original = &Error{
		Errors: []error{errors.New("foo")},
	}

	result = Append(original, Append(nil, errors.New("foo"), errors.New("bar")))
	if len(result.Errors) != 3 {
		t.Fatalf("wrong len: %d", len(result.Errors))
	}
}

func TestAppend_NilError(t *testing.T) {
	var err error
	result := Append(err, errors.New("bar"))
	if len(result.Errors) != 1 {
		t.Fatalf("wrong len: %d", len(result.Errors))
	}
}

func TestAppend_NonError(t *testing.T) {
	original := errors.New("foo")
	result := Append(original, errors.New("bar"))
	if len(result.Errors) != 2 {
		t.Fatalf("wrong len: %d", len(result.Errors))
	}
}

func TestAppend_NonError_Error(t *testing.T) {
	original := errors.New("foo")
	result := Append(original, Append(nil, errors.New("bar")))
	if len(result.Errors) != 2 {
		t.Fatalf("wrong len: %d", len(result.Errors))
	}
}
