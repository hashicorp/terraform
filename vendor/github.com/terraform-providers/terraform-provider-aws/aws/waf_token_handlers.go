package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
)

type WafRetryer struct {
	Connection *waf.WAF
	Region     string
}

type withTokenFunc func(token *string) (interface{}, error)

func (t *WafRetryer) RetryWithToken(f withTokenFunc) (interface{}, error) {
	awsMutexKV.Lock(t.Region)
	defer awsMutexKV.Unlock(t.Region)

	var out interface{}
	err := resource.Retry(15*time.Minute, func() *resource.RetryError {
		var err error
		var tokenOut *waf.GetChangeTokenOutput

		tokenOut, err = t.Connection.GetChangeToken(&waf.GetChangeTokenInput{})
		if err != nil {
			return resource.NonRetryableError(errwrap.Wrapf("Failed to acquire change token: {{err}}", err))
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

func newWafRetryer(conn *waf.WAF, region string) *WafRetryer {
	return &WafRetryer{Connection: conn, Region: region}
}
