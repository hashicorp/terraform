package projects

import (
	"sort"

	"github.com/hashicorp/terraform/addrs"
)

type sortWorkspaceAddrs []addrs.ProjectWorkspace

var _ sort.Interface = sortWorkspaceAddrs(nil)

func (s sortWorkspaceAddrs) Len() int {
	return len(s)
}

func (s sortWorkspaceAddrs) Less(i, j int) bool {
	switch {
	case s[i].Rel != s[j].Rel:
		return s[i].Rel == addrs.ProjectWorkspaceCurrent
	case s[i].Name != s[j].Name:
		return s[i].Name < s[j].Name
	case s[i].Key != s[j].Key:
		return addrs.InstanceKeyLess(s[i].Key, s[j].Key)
	default:
		return false
	}
}

func (s sortWorkspaceAddrs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
