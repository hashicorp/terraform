package models

import "errors"

type Role int

const (
	RoleUnknown Role = iota - 1
	RoleOrgUser
	RoleOrgManager
	RoleBillingManager
	RoleOrgAuditor
	RoleSpaceManager
	RoleSpaceDeveloper
	RoleSpaceAuditor
)

var ErrUnknownRole = errors.New("Unknown Role")

func RoleFromString(roleString string) (Role, error) {
	switch roleString {
	case "OrgManager":
		return RoleOrgManager, nil
	case "BillingManager":
		return RoleBillingManager, nil
	case "OrgAuditor":
		return RoleOrgAuditor, nil
	case "SpaceManager":
		return RoleSpaceManager, nil
	case "SpaceDeveloper":
		return RoleSpaceDeveloper, nil
	case "SpaceAuditor":
		return RoleSpaceAuditor, nil
	default:
		return RoleUnknown, ErrUnknownRole
	}
}

func (r Role) ToString() string {
	switch r {
	case RoleUnknown:
		return "RoleUnknown"
	case RoleOrgUser:
		return "RoleOrgUser"
	case RoleOrgManager:
		return "RoleOrgManager"
	case RoleBillingManager:
		return "RoleBillingManager"
	case RoleOrgAuditor:
		return "RoleOrgAuditor"
	case RoleSpaceManager:
		return "RoleSpaceManager"
	case RoleSpaceDeveloper:
		return "RoleSpaceDeveloper"
	case RoleSpaceAuditor:
		return "RoleSpaceAuditor"
	default:
		return ""
	}
}
