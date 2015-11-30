package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/hashicorp/terraform/helper/resource"
)

// Better waiter for AWS, extending the core waiter logic.
//
// Example:
//     params := &awsservice.APICallInput { Name: value }
//     out, err := retryWaiter(
//       func() (interface{}, error) { return conn.APIFunction(params) },
//       []string{"ListOfRetryableErrors"},
//       1*time.Minute, // or other valid time representation
//     )
//
//     if err != nil {
//       // handle errors here
//     }
//
//     apiCallOutput := out.(*service.APICallOutput)
func retryWaiter(f func() (interface{}, error), codes []string, timeout time.Duration) (interface{}, error) {
	var output interface{}

	err := resource.Retry(timeout, func() error {
		var err error
		output, err = f()

		if err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				for _, code := range codes {
					if code == awserr.Code() {
						// Retryable
						return awserr
					}
				}
				// Didn't recognize the error, so shouldn't retry.
				return resource.RetryError{Err: err}
			}
		}
		// Successful
		return nil
	})
	return output, err
}
