package github

import "errors"

const pullPermission string = "pull"
const pushPermission string = "push"
const adminPermission string = "admin"

func getRepoPermission(p *map[string]bool) (string, error) {

	// Permissions are returned in this map format such that if you have a certain level
	// of permission, all levels below are also true. For example, if a team has push
	// permission, the map will be: {"pull": true, "push": true, "admin": false}
	if (*p)[adminPermission] {
		return adminPermission, nil
	} else if (*p)[pushPermission] {
		return pushPermission, nil
	} else {
		if (*p)[pullPermission] {
			return pullPermission, nil
		}
		return "", errors.New("At least one permission expected from permissions map.")
	}
}
