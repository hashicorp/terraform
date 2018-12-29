package aws

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/hashicorp/terraform/helper/resource"
)

// Returns true if the error matches all these conditions:
//  * err is of type awserr.Error
//  * Error.Code() matches code
//  * Error.Message() contains message
func isAWSErr(err error, code string, message string) bool {
	if err, ok := err.(awserr.Error); ok {
		return err.Code() == code && strings.Contains(err.Message(), message)
	}
	return false
}

// IsAWSErrExtended returns true if the error matches all conditions
//  * err is of type awserr.Error
//  * Error.Code() matches code
//  * Error.Message() contains message
//  * Error.OrigErr() contains origErrMessage
// Note: This function will be moved out of the aws package in the future.
func IsAWSErrExtended(err error, code string, message string, origErrMessage string) bool {
	if !isAWSErr(err, code, message) {
		return false
	}
	return strings.Contains(err.(awserr.Error).OrigErr().Error(), origErrMessage)
}

func retryOnAwsCode(code string, f func() (interface{}, error)) (interface{}, error) {
	var resp interface{}
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		resp, err = f()
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok && awsErr.Code() == code {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	return resp, err
}

// RetryOnAwsCodes retries AWS error codes for one minute
// Note: This function will be moved out of the aws package in the future.
func RetryOnAwsCodes(codes []string, f func() (interface{}, error)) (interface{}, error) {
	var resp interface{}
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		resp, err = f()
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok {
				for _, code := range codes {
					if awsErr.Code() == code {
						return resource.RetryableError(err)
					}
				}
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	return resp, err
}
