package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// GraphNodeAttachProvider is an interface that must be implemented by nodes
// that want provider configurations attached.
type GraphNodeAttachProvider interface {
	// Must be implemented to determine the path for the configuration
	GraphNodeSubPath

	// ProviderName with no module prefix. Example: "aws".
	ProviderName() string

	// Sets the configuration
	AttachProvider(*config.ProviderConfig)
}
