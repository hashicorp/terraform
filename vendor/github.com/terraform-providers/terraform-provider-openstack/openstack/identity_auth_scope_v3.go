package openstack

import (
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
)

func flattenIdentityAuthScopeV3Roles(roles []tokens.Role) []map[string]string {
	allRoles := make([]map[string]string, len(roles))

	for i, r := range roles {
		allRoles[i] = map[string]string{
			"role_name": r.Name,
			"role_id":   r.ID,
		}

	}

	return allRoles
}
