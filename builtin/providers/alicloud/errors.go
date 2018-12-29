package alicloud

const (
	// common
	Notfound = "Not found"
	// ecs
	InstanceNotfound = "Instance.Notfound"
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

	// vswitch
	VswitcInvalidRegionId = "InvalidRegionId.NotFound"
)
