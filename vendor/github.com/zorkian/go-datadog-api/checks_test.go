package datadog_test

import (
	"testing"

	"github.com/zorkian/go-datadog-api"
)

func TestCheckStatus(T *testing.T) {
	if datadog.OK != 0 {
		T.Error("status OK must be 0 to satisfy Datadog's API")
	}
	if datadog.WARNING != 1 {
		T.Error("status WARNING must be 1 to satisfy Datadog's API")
	}
	if datadog.CRITICAL != 2 {
		T.Error("status CRITICAL must be 2 to satisfy Datadog's API")
	}
	if datadog.UNKNOWN != 3 {
		T.Error("status UNKNOWN must be 3 to satisfy Datadog's API")
	}
}
