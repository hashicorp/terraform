package getproviders

import (
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform/addrs"
)

// HTTPMirrorSource is a source that reads provider metadata from a provider
// mirror that is accessible over the HTTP provider mirror protocol.
type HTTPMirrorSource struct {
	baseURL *url.URL
}

var _ Source = (*HTTPMirrorSource)(nil)

// NewHTTPMirrorSource constructs and returns a new network mirror source with
// the given base URL. The relative URL offsets defined by the HTTP mirror
// protocol will be resolve relative to the given URL.
func NewHTTPMirrorSource(baseURL *url.URL) *HTTPMirrorSource {
	return &HTTPMirrorSource{
		baseURL: baseURL,
	}
}

// AvailableVersions retrieves the available versions for the given provider
// from the object's underlying HTTP mirror service.
func (s *HTTPMirrorSource) AvailableVersions(provider addrs.Provider) (VersionList, error) {
	return nil, fmt.Errorf("Network-based provider mirrors are not supported in this version of Terraform")
}

// PackageMeta retrieves metadata for the requested provider package
// from the object's underlying HTTP mirror service.
func (s *HTTPMirrorSource) PackageMeta(provider addrs.Provider, version Version, target Platform) (PackageMeta, error) {
	return PackageMeta{}, fmt.Errorf("Network-based provider mirrors are not supported in this version of Terraform")
}
