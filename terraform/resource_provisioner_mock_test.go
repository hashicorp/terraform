package terraform

import (
	"testing"
)

func TestMockResourceProvisioner_impl(t *testing.T) {
	var _ ResourceProvisioner = new(MockResourceProvisioner)
}
