package terraform

import (
	"github.com/hashicorp/terraform/version"
)

// TODO: update providers to use the version package directly
func VersionString() string {
	return version.String()
}
