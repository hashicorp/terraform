// Package errors implements functions for manipulating errors.
//
// The traditional error handling idiom in Go is roughly akin to
//
//      if err != nil {
//              return err
//      }
//
// which applied recursively up the call stack results in error reports
// without context or debugging information. The errors package allows
// programmers to add context to the failure path in their code in a way
// that does not destroy the original value of the error.
//
// Adding context to an error
//
// The errors.Wrap function returns a new error that adds context to the
// original error. For example
//
//      _, err := ioutil.ReadAll(r)
//      if err != nil {
//              return errors.Wrap(err, "read failed")
//      }
//
// In addition, errors.Wrap records the file and line where it was called,
// allowing the programmer to retrieve the path to the original error.
//
// Retrieving the cause of an error
//
// Using errors.Wrap constructs a stack of errors, adding context to the
// preceding error. Depending on the nature of the error it may be necessary
// to reverse the operation of errors.Wrap to retrieve the original error
// for inspection. Any error value which implements this interface
//
//     type causer interface {
//          Cause() error
//     }
//
// can be inspected by errors.Cause. errors.Cause will recursively retrieve
// the topmost error which does nor implement causer, which is assumed to be
// the original cause. For example:
//
//     switch err := errors.Cause(err).(type) {
//     case *MyError:
//             // handle specifically
//     default:
//             // unknown error
//     }
package errors

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

// location represents a program counter that
// implements the Location() method.
type location uintptr

func (l location) Location() (string, int) {
	pc := uintptr(l) - 1
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown", 0
	}

	file, line := fn.FileLine(pc)

	// Here we want to get the source file path relative to the compile time
	// GOPATH. As of Go 1.6.x there is no direct way to know the compiled
	// GOPATH at runtime, but we can infer the number of path segments in the
	// GOPATH. We note that fn.Name() returns the function name qualified by
	// the import path, which does not include the GOPATH. Thus we can trim
	// segments from the beginning of the file path until the number of path
	// separators remaining is one more than the number of path separators in
	// the function name. For example, given:
	//
	//    GOPATH     /home/user
	//    file       /home/user/src/pkg/sub/file.go
	//    fn.Name()  pkg/sub.Type.Method
	//
	// We want to produce:
	//
	//    pkg/sub/file.go
	//
	// From this we can easily see that fn.Name() has one less path separator
	// than our desired output. We count separators from the end of the file
	// path until it finds two more than in the function name and then move
	// one character forward to preserve the initial path segment without a
	// leading separator.
	const sep = "/"
	goal := strings.Count(fn.Name(), sep) + 2
	i := len(file)
	for n := 0; n < goal; n++ {
		i = strings.LastIndex(file[:i], sep)
		if i == -1 {
			// not enough separators found, set i so that the slice expression
			// below leaves file unmodified
			i = -len(sep)
			break
		}
	}
	// get back to 0 or trim the leading separator
	file = file[i+len(sep):]

	return file, line
}

// New returns an error that formats as the given text.
func New(text string) error {
	pc, _, _, _ := runtime.Caller(1)
	return struct {
		error
		location
	}{
		errors.New(text),
		location(pc),
	}
}

type cause struct {
	cause   error
	message string
}

func (c cause) Error() string   { return c.Message() + ": " + c.Cause().Error() }
func (c cause) Cause() error    { return c.cause }
func (c cause) Message() string { return c.message }

// Errorf formats according to a format specifier and returns the string
// as a value that satisfies error.
func Errorf(format string, args ...interface{}) error {
	pc, _, _, _ := runtime.Caller(1)
	return struct {
		error
		location
	}{
		fmt.Errorf(format, args...),
		location(pc),
	}
}

// Wrap returns an error annotating the cause with message.
// If cause is nil, Wrap returns nil.
func Wrap(cause error, message string) error {
	if cause == nil {
		return nil
	}
	pc, _, _, _ := runtime.Caller(1)
	return wrap(cause, message, pc)
}

// Wrapf returns an error annotating the cause with the format specifier.
// If cause is nil, Wrapf returns nil.
func Wrapf(cause error, format string, args ...interface{}) error {
	if cause == nil {
		return nil
	}
	pc, _, _, _ := runtime.Caller(1)
	return wrap(cause, fmt.Sprintf(format, args...), pc)
}

func wrap(err error, msg string, pc uintptr) error {
	return struct {
		cause
		location
	}{
		cause{
			cause:   err,
			message: msg,
		},
		location(pc),
	}
}

type causer interface {
	Cause() error
}

// Cause returns the underlying cause of the error, if possible.
// An error value has a cause if it implements the following
// interface:
//
//     type Causer interface {
//            Cause() error
//     }
//
// If the error does not implement Cause, the original error will
// be returned. If the error is nil, nil will be returned without further
// investigation.
func Cause(err error) error {
	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}
	return err
}

// Print prints the error to Stderr.
// If the error implements the Causer interface described in Cause
// Print will recurse into the error's cause.
// If the error implements the inteface:
//
//     type Location interface {
//            Location() (file string, line int)
//     }
//
// Print will also print the file and line of the error.
func Print(err error) {
	Fprint(os.Stderr, err)
}

// Fprint prints the error to the supplied writer.
// The format of the output is the same as Print.
// If err is nil, nothing is printed.
func Fprint(w io.Writer, err error) {
	type location interface {
		Location() (string, int)
	}
	type message interface {
		Message() string
	}

	for err != nil {
		if err, ok := err.(location); ok {
			file, line := err.Location()
			fmt.Fprintf(w, "%s:%d: ", file, line)
		}
		switch err := err.(type) {
		case message:
			fmt.Fprintln(w, err.Message())
		default:
			fmt.Fprintln(w, err.Error())
		}

		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}
}
