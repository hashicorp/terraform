package errors

const (
	MessageParseError                      = "1001"
	InvalidRelation                        = "1002"
	NotAuthorized                          = "10003"
	BadQueryParameter                      = "10005"
	UserNotFound                           = "20003"
	OrganizationNameTaken                  = "30002"
	SpaceNameTaken                         = "40002"
	ServiceInstanceNameTaken               = "60002"
	ServiceBindingAppServiceTaken          = "90003"
	UnbindableService                      = "90005"
	ServiceInstanceAlreadyBoundToSameRoute = "130008"
	NotStaged                              = "170002"
	InstancesError                         = "220001"
	QuotaDefinitionNameTaken               = "240002"
	BuildpackNameTaken                     = "290001"
	SecurityGroupNameTaken                 = "300005"
	ServiceKeyNameTaken                    = "360001"
)
