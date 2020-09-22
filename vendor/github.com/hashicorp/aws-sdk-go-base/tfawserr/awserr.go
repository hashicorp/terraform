package tfawserr

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
)

// ErrMessageAndOrigErrContain returns true if the error matches all these conditions:
//  * err is of type awserr.Error
//  * Error.Code() matches code
//  * Error.Message() contains message
//  * Error.OrigErr() contains origErrMessage
func ErrMessageAndOrigErrContain(err error, code string, message string, origErrMessage string) bool {
	if !ErrMessageContains(err, code, message) {
		return false
	}

	if origErrMessage == "" {
		return true
	}

	// Ensure OrigErr() is non-nil, to prevent panics
	if origErr := err.(awserr.Error).OrigErr(); origErr != nil {
		return strings.Contains(origErr.Error(), origErrMessage)
	}

	return false
}

// ErrCodeEquals returns true if the error matches all these conditions:
//  * err is of type awserr.Error
//  * Error.Code() equals code
func ErrCodeEquals(err error, code string) bool {
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		return awsErr.Code() == code
	}
	return false
}

// ErrCodeContains returns true if the error matches all these conditions:
//  * err is of type awserr.Error
//  * Error.Code() contains code
func ErrCodeContains(err error, code string) bool {
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		return strings.Contains(awsErr.Code(), code)
	}
	return false
}

// ErrMessageContains returns true if the error matches all these conditions:
//  * err is of type awserr.Error
//  * Error.Code() equals code
//  * Error.Message() contains message
func ErrMessageContains(err error, code string, message string) bool {
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		return awsErr.Code() == code && strings.Contains(awsErr.Message(), message)
	}
	return false
}

// ErrStatusCodeEquals returns true if the error matches all these conditions:
//  * err is of type awserr.RequestFailure
//  * RequestFailure.StatusCode() equals statusCode
// It is always preferable to use ErrMessageContains() except in older APIs (e.g. S3)
// that sometimes only respond with status codes.
func ErrStatusCodeEquals(err error, statusCode int) bool {
	var awsErr awserr.RequestFailure
	if errors.As(err, &awsErr) {
		return awsErr.StatusCode() == statusCode
	}
	return false
}
