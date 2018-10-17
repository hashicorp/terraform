package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
)

// GraphNodeAttachProvider is an interface that must be implemented by nodes
// that want provider configurations attached.
type GraphNodeAttachProvider interface {
	// Must be implemented to determine the path for the configuration
	GraphNodeSubPath

	// ProviderName with no module prefix. Example: "aws".
	ProviderAddr() addrs.AbsProviderConfig

	// Sets the configuration
	AttachProvider(*configs.Provider)
}
