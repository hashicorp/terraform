package terraform

import (
	"testing"
)

func TestMockResourceProvider_impl(t *testing.T) {
	var _ ResourceProvider = new(MockResourceProvider)
	var _ ResourceProviderCloser = new(MockResourceProvider)
}
