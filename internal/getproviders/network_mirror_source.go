package getproviders

import (
	"fmt"

	svchost "github.com/hashicorp/terraform-svchost"

	"github.com/hashicorp/terraform/addrs"
)

// NetworkMirrorSource is a source that reads providers and their metadata
// from an HTTP server implementing the Terraform network mirror protocol.
type NetworkMirrorSource struct {
	host svchost.Hostname
}

var _ Source = (*NetworkMirrorSource)(nil)

// NewNetworkMirrorSource constructs and returns a new network-based
// mirror source that will expect to find a mirror service on the given
// host.
func NewNetworkMirrorSource(host svchost.Hostname) *NetworkMirrorSource {
	return &NetworkMirrorSource{
		host: host,
	}
}

// AvailableVersions retrieves the available versions for the given provider
// from the network mirror.
func (s *NetworkMirrorSource) AvailableVersions(provider addrs.Provider) (VersionList, error) {
	return nil, fmt.Errorf("Network provider mirror is not supported in this version of Terraform")
}

// PackageMeta checks to see if the network mirror contains a copy of the
// distribution package for the given provider version on the given target,
// and returns the metadata about it if so.
func (s *NetworkMirrorSource) PackageMeta(provider addrs.Provider, version Version, target Platform) (PackageMeta, error) {
	return PackageMeta{}, fmt.Errorf("Network provider mirror is not supported in this version of Terraform")
}
