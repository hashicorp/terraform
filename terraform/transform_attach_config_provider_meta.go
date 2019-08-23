package terraform

import (
	"github.com/hashicorp/terraform/configs"
)

// GraphNodeAttachProviderMetaConfigs is an interface that must be implemented
// by nodes that want provider meta configurations attached.
type GraphNodeAttachProviderMetaConfigs interface {
	GraphNodeResource

	// Sets the configuration
	AttachProviderMetaConfigs(*configs.ProviderMeta)
}
