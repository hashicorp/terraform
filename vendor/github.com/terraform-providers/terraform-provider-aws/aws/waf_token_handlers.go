package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/resource"
)

type WafRetryer struct {
	Connection *waf.WAF
}

type withTokenFunc func(token *string) (interface{}, error)

func (t *WafRetryer) RetryWithToken(f withTokenFunc) (interface{}, error) {
	awsMutexKV.Lock("WafRetryer")
	defer awsMutexKV.Unlock("WafRetryer")

	var out interface{}
	err := resource.Retry(15*time.Minute, func() *resource.RetryError {
		var err error
		var tokenOut *waf.GetChangeTokenOutput

		tokenOut, err = t.Connection.GetChangeToken(&waf.GetChangeTokenInput{})
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("Failed to acquire change token: %s", err))
		}

		out, err = f(tokenOut.ChangeToken)
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok && awsErr.Code() == "WAFStaleDataException" {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	return out, err
}

func newWafRetryer(conn *waf.WAF) *WafRetryer {
	return &WafRetryer{Connection: conn}
}
