package providermocks

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
)

// ResourceType is an address type identifying a particular resource type
// offered by a provider.
type ResourceType struct {
	Mode addrs.ResourceMode
	Type string
}

func (addr ResourceType) String() string {
	switch addr.Mode {
	case addrs.ManagedResourceMode:
		return addr.Type
	case addrs.DataResourceMode:
		return "data." + addr.Type
	default:
		panic(fmt.Sprintf("ResourceType with unrecognized mode %s", addr.Mode))
	}
}

func (addr ResourceType) MockConfigFilename() string {
	switch addr.Mode {
	case addrs.ManagedResourceMode:
		return "resource." + addr.Type + mockFilenameSuffix
	case addrs.DataResourceMode:
		return "data." + addr.Type + mockFilenameSuffix
	default:
		panic(fmt.Sprintf("ResourceType with unrecognized mode %s", addr.Mode))
	}
}
