package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
)

type WafRegionalRetryer struct {
	Connection *wafregional.WAFRegional
	Region     string
}

type withRegionalTokenFunc func(token *string) (interface{}, error)

func (t *WafRegionalRetryer) RetryWithToken(f withRegionalTokenFunc) (interface{}, error) {
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

func newWafRegionalRetryer(conn *wafregional.WAFRegional, region string) *WafRegionalRetryer {
	return &WafRegionalRetryer{Connection: conn, Region: region}
}
