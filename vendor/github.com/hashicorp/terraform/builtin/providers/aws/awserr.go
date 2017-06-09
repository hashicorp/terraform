package aws

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
)

func isAWSErr(err error, code string, message string) bool {
	if err, ok := err.(awserr.Error); ok {
		return err.Code() == code && strings.Contains(err.Message(), message)
	}
	return false
}
