package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// NodeApplyableProvider represents a provider during an apply.
//
// NOTE: There is a lot of logic here that will be shared with non-Apply.
// The plan is to abstract that eventually into an embedded abstract struct.
type NodeApplyableProvider struct {
	NameValue string
	PathValue []string
	Config    *config.ProviderConfig
}

func (n *NodeApplyableProvider) Name() string {
	result := fmt.Sprintf("provider.%s", n.NameValue)
	if len(n.PathValue) > 1 {
		result = fmt.Sprintf("%s.%s", modulePrefixStr(n.PathValue), result)
	}

	return result
}

// GraphNodeSubPath
func (n *NodeApplyableProvider) Path() []string {
	return n.PathValue
}

// GraphNodeProvider
func (n *NodeApplyableProvider) ProviderName() string {
	return n.NameValue
}

// GraphNodeProvider
func (n *NodeApplyableProvider) ProviderConfig() *config.RawConfig {
	if n.Config == nil {
		return nil
	}

	return n.Config.RawConfig
}

// GraphNodeAttachProvider
func (n *NodeApplyableProvider) AttachProvider(c *config.ProviderConfig) {
	n.Config = c
}

// GraphNodeEvalable
func (n *NodeApplyableProvider) EvalTree() EvalNode {
	return ProviderEvalTree(n.NameValue, nil)
}
