package awsbase

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
)

// IsAWSErr returns true if the error matches all these conditions:
//  * err is of type awserr.Error
//  * Error.Code() matches code
//  * Error.Message() contains message
func IsAWSErr(err error, code string, message string) bool {
	var awsErr awserr.Error

	if errors.As(err, &awsErr) {
		return awsErr.Code() == code && strings.Contains(awsErr.Message(), message)
	}

	return false
}

// IsAWSErrExtended returns true if the error matches all these conditions:
//  * err is of type awserr.Error
//  * Error.Code() matches code
//  * Error.Message() contains message
//  * Error.OrigErr() contains origErrMessage
func IsAWSErrExtended(err error, code string, message string, origErrMessage string) bool {
	if !IsAWSErr(err, code, message) {
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
