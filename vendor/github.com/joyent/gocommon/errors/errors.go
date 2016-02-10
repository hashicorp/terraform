//
// gocommon - Go library to interact with the JoyentCloud
// This package provides an Error implementation which knows about types of error, and which has support
// for error causes.
//
// Copyright (c) 2013 Joyent Inc.
//
// Written by Daniele Stroppa <daniele.stroppa@joyent.com>
//

package errors

import "fmt"

type Code string

const (
	// Public available error types.
	// These errors are provided because they are specifically required by business logic in the callers.
	BadRequestError         = Code("BadRequest")
	InternalErrorError      = Code("InternalError")
	InvalidArgumentError    = Code("InvalidArgument")
	InvalidCredentialsError = Code("InvalidCredentials")
	InvalidHeaderError      = Code("InvalidHeader")
	InvalidVersionError     = Code("InvalidVersion")
	MissingParameterError   = Code("MissinParameter")
	NotAuthorizedError      = Code("NotAuthorized")
	RequestThrottledError   = Code("RequestThrottled")
	RequestTooLargeError    = Code("RequestTooLarge")
	RequestMovedError       = Code("RequestMoved")
	ResourceNotFoundError   = Code("ResourceNotFound")
	UnknownErrorError       = Code("UnkownError")
)

// Error instances store an optional error cause.
type Error interface {
	error
	Cause() error
}

type gojoyentError struct {
	error
	errcode Code
	cause   error
}

// Type checks.
var _ Error = (*gojoyentError)(nil)

// Code returns the error code.
func (err *gojoyentError) code() Code {
	if err.errcode != UnknownErrorError {
		return err.errcode
	}
	if e, ok := err.cause.(*gojoyentError); ok {
		return e.code()
	}
	return UnknownErrorError
}

// Cause returns the error cause.
func (err *gojoyentError) Cause() error {
	return err.cause
}

// CausedBy returns true if this error or its cause are of the specified error code.
func (err *gojoyentError) causedBy(code Code) bool {
	if err.code() == code {
		return true
	}
	if cause, ok := err.cause.(*gojoyentError); ok {
		return cause.code() == code
	}
	return false
}

// Error fulfills the error interface, taking account of any caused by error.
func (err *gojoyentError) Error() string {
	if err.cause != nil {
		return fmt.Sprintf("%v\ncaused by: %v", err.error, err.cause)
	}
	return err.error.Error()
}

func IsBadRequest(err error) bool {
	if e, ok := err.(*gojoyentError); ok {
		return e.causedBy(BadRequestError)
	}
	return false
}

func IsInternalError(err error) bool {
	if e, ok := err.(*gojoyentError); ok {
		return e.causedBy(InternalErrorError)
	}
	return false
}

func IsInvalidArgument(err error) bool {
	if e, ok := err.(*gojoyentError); ok {
		return e.causedBy(InvalidArgumentError)
	}
	return false
}

func IsInvalidCredentials(err error) bool {
	if e, ok := err.(*gojoyentError); ok {
		return e.causedBy(InvalidCredentialsError)
	}
	return false
}

func IsInvalidHeader(err error) bool {
	if e, ok := err.(*gojoyentError); ok {
		return e.causedBy(InvalidHeaderError)
	}
	return false
}

func IsInvalidVersion(err error) bool {
	if e, ok := err.(*gojoyentError); ok {
		return e.causedBy(InvalidVersionError)
	}
	return false
}

func IsMissingParameter(err error) bool {
	if e, ok := err.(*gojoyentError); ok {
		return e.causedBy(MissingParameterError)
	}
	return false
}

func IsNotAuthorized(err error) bool {
	if e, ok := err.(*gojoyentError); ok {
		return e.causedBy(NotAuthorizedError)
	}
	return false
}

func IsRequestThrottled(err error) bool {
	if e, ok := err.(*gojoyentError); ok {
		return e.causedBy(RequestThrottledError)
	}
	return false
}

func IsRequestTooLarge(err error) bool {
	if e, ok := err.(*gojoyentError); ok {
		return e.causedBy(RequestTooLargeError)
	}
	return false
}

func IsRequestMoved(err error) bool {
	if e, ok := err.(*gojoyentError); ok {
		return e.causedBy(RequestMovedError)
	}
	return false
}

func IsResourceNotFound(err error) bool {
	if e, ok := err.(*gojoyentError); ok {
		return e.causedBy(ResourceNotFoundError)
	}
	return false
}

func IsUnknownError(err error) bool {
	if e, ok := err.(*gojoyentError); ok {
		return e.causedBy(UnknownErrorError)
	}
	return false
}

// New creates a new Error instance with the specified cause.
func makeErrorf(code Code, cause error, format string, args ...interface{}) Error {
	return &gojoyentError{
		errcode: code,
		error:   fmt.Errorf(format, args...),
		cause:   cause,
	}
}

// New creates a new UnknownError Error instance with the specified cause.
func Newf(cause error, format string, args ...interface{}) Error {
	return makeErrorf(UnknownErrorError, cause, format, args...)
}

// New creates a new BadRequest Error instance with the specified cause.
func NewBadRequestf(cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Bad Request: %s", context)
	}
	return makeErrorf(BadRequestError, cause, format, args...)
}

// New creates a new InternalError Error instance with the specified cause.
func NewInternalErrorf(cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Internal Error: %s", context)
	}
	return makeErrorf(InternalErrorError, cause, format, args...)
}

// New creates a new InvalidArgument Error instance with the specified cause.
func NewInvalidArgumentf(cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Invalid Argument: %s", context)
	}
	return makeErrorf(InvalidArgumentError, cause, format, args...)
}

// New creates a new InvalidCredentials Error instance with the specified cause.
func NewInvalidCredentialsf(cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Invalid Credentials: %s", context)
	}
	return makeErrorf(InvalidCredentialsError, cause, format, args...)
}

// New creates a new InvalidHeader Error instance with the specified cause.
func NewInvalidHeaderf(cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Invalid Header: %s", context)
	}
	return makeErrorf(InvalidHeaderError, cause, format, args...)
}

// New creates a new InvalidVersion Error instance with the specified cause.
func NewInvalidVersionf(cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Invalid Version: %s", context)
	}
	return makeErrorf(InvalidVersionError, cause, format, args...)
}

// New creates a new MissingParameter Error instance with the specified cause.
func NewMissingParameterf(cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Missing Parameter: %s", context)
	}
	return makeErrorf(MissingParameterError, cause, format, args...)
}

// New creates a new NotAuthorized Error instance with the specified cause.
func NewNotAuthorizedf(cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Not Authorized: %s", context)
	}
	return makeErrorf(NotAuthorizedError, cause, format, args...)
}

// New creates a new RequestThrottled Error instance with the specified cause.
func NewRequestThrottledf(cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Request Throttled: %s", context)
	}
	return makeErrorf(RequestThrottledError, cause, format, args...)
}

// New creates a new RequestTooLarge Error instance with the specified cause.
func NewRequestTooLargef(cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Request Too Large: %s", context)
	}
	return makeErrorf(RequestTooLargeError, cause, format, args...)
}

// New creates a new RequestMoved Error instance with the specified cause.
func NewRequestMovedf(cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Request Moved: %s", context)
	}
	return makeErrorf(RequestMovedError, cause, format, args...)
}

// New creates a new ResourceNotFound Error instance with the specified cause.
func NewResourceNotFoundf(cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Resource Not Found: %s", context)
	}
	return makeErrorf(ResourceNotFoundError, cause, format, args...)
}

// New creates a new UnknownError Error instance with the specified cause.
func NewUnknownErrorf(cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Unknown Error: %s", context)
	}
	return makeErrorf(UnknownErrorError, cause, format, args...)
}
