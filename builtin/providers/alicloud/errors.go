package alicloud

import (
	"github.com/denverdino/aliyungo/common"
	"strings"
)

const (
	// common
	Notfound = "Not found"
	// ecs
	InstanceNotFound        = "Instance.Notfound"
	MessageInstanceNotFound = "instance is not found"
	// disk
	DiskIncorrectStatus       = "IncorrectDiskStatus"
	DiskCreatingSnapshot      = "DiskCreatingSnapshot"
	InstanceLockedForSecurity = "InstanceLockedForSecurity"
	SystemDiskNotFound        = "SystemDiskNotFound"
	// eip
	EipIncorrectStatus      = "IncorrectEipStatus"
	InstanceIncorrectStatus = "IncorrectInstanceStatus"
	HaVipIncorrectStatus    = "IncorrectHaVipStatus"
	// slb
	LoadBalancerNotFound = "InvalidLoadBalancerId.NotFound"

	// security_group
	InvalidInstanceIdAlreadyExists = "InvalidInstanceId.AlreadyExists"
	InvalidSecurityGroupIdNotFound = "InvalidSecurityGroupId.NotFound"
	SgDependencyViolation          = "DependencyViolation"

	//Nat gateway
	NatGatewayInvalidRegionId            = "Invalid.RegionId"
	DependencyViolationBandwidthPackages = "DependencyViolation.BandwidthPackages"
	NotFindSnatEntryBySnatId             = "NotFindSnatEntryBySnatId"
	NotFindForwardEntryByForwardId       = "NotFindForwardEntryByForwardId"

	// vswitch
	VswitcInvalidRegionId = "InvalidRegionId.NotFound"

	// ess
	InvalidScalingGroupIdNotFound               = "InvalidScalingGroupId.NotFound"
	IncorrectScalingConfigurationLifecycleState = "IncorrectScalingConfigurationLifecycleState"

	// oss
	OssBucketNotFound = "NoSuchBucket"

	// RAM Instance Not Found
	RamInstanceNotFound   = "Forbidden.InstanceNotFound"
	AliyunGoClientFailure = "AliyunGoClientFailure"

	//unknown Error
	UnknownError = "UnknownError"
)

func GetNotFoundErrorFromString(str string) error {
	return &common.Error{
		ErrorResponse: common.ErrorResponse{
			Code:    InstanceNotFound,
			Message: str,
		},
		StatusCode: -1,
	}
}

func NotFoundError(err error) bool {
	if e, ok := err.(*common.Error); ok &&
		(e.Code == InstanceNotFound || e.Code == RamInstanceNotFound ||
			strings.Contains(strings.ToLower(e.Message), MessageInstanceNotFound)) {
		return true
	}

	return false
}
