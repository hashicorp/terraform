package github

import (
	"errors"
	"fmt"

	"github.com/google/go-github/github"
)

const (
	pullPermission  string = "pull"
	pushPermission  string = "push"
	adminPermission string = "admin"

	writePermission string = "write"
	readPermission  string = "read"
)

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

func getInvitationPermission(i *github.RepositoryInvitation) (string, error) {
	// Permissions for some GitHub API routes are expressed as "read",
	// "write", and "admin"; in other places, they are expressed as "pull",
	// "push", and "admin".
	if *i.Permissions == readPermission {
		return pullPermission, nil
	} else if *i.Permissions == writePermission {
		return pushPermission, nil
	} else if *i.Permissions == adminPermission {
		return adminPermission, nil
	}

	return "", fmt.Errorf("unexpected permission value: %v", *i.Permissions)
}
