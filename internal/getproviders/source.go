package getproviders

import (
	"github.com/hashicorp/terraform/addrs"
)

// A Source can query a particular source for information about providers
// that are available to install.
type Source interface {
	AvailableVersions(provider addrs.Provider) (VersionList, Warnings, error)
	PackageMeta(provider addrs.Provider, version Version, target Platform) (PackageMeta, error)
	ForDisplay(provider addrs.Provider) string
}
