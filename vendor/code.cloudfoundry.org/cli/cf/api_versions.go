package cf

import "github.com/blang/semver"

var (
	ReservedRoutePortsMinimumAPIVersion, _              = semver.Make("2.55.0") // #112023051
	TCPRoutingMinimumAPIVersion, _                      = semver.Make("2.53.0") // #111475922
	MultipleAppPortsMinimumAPIVersion, _                = semver.Make("2.51.0")
	SpaceAppInstanceLimitMinimumAPIVersion, _           = semver.Make("2.40.0")
	SetRolesByUsernameMinimumAPIVersion, _              = semver.Make("2.37.0")
	RoutePathMinimumAPIVersion, _                       = semver.Make("2.36.0")
	OrgAppInstanceLimitMinimumAPIVersion, _             = semver.Make("2.33.0")
	NoaaMinimumAPIVersion, _                            = semver.Make("2.29.0")
	ListUsersInOrgOrSpaceWithoutUAAMinimumAPIVersion, _ = semver.Make("2.21.0")
	UpdateServicePlanMinimumAPIVersion, _               = semver.Make("2.16.0")

	ServiceAuthTokenMaximumAPIVersion, _ = semver.Make("2.46.0")
	SpaceScopedMaximumAPIVersion, _      = semver.Make("2.47.0")
)
