package alicloud

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"strings"
)

type LifecycleRuleStatus string

const (
	ExpirationStatusEnabled  = LifecycleRuleStatus("Enabled")
	ExpirationStatusDisabled = LifecycleRuleStatus("Disabled")
)

func ossNotFoundError(err error) bool {
	if e, ok := err.(oss.ServiceError); ok &&
		(e.StatusCode == 404 || strings.HasPrefix(e.Code, "NoSuch") || strings.HasPrefix(e.Message, "No Row found")) {
		return true
	}
	return false
}
