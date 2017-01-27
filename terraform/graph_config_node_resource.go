package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// GraphNodeConfigResource represents a resource within the config graph.
type GraphNodeConfigResource struct {
	Resource *config.Resource

	// If set to true, this resource represents a resource
	// that will be destroyed in some way.
	Destroy bool

	// Used during DynamicExpand to target indexes
	Targets []ResourceAddress

	Path []string
}

func (n *GraphNodeConfigResource) DependableName() []string {
	return []string{n.Resource.Id()}
}

// GraphNodeDependent impl.
func (n *GraphNodeConfigResource) DependentOn() []string {
	result := make([]string, len(n.Resource.DependsOn),
		(len(n.Resource.RawCount.Variables)+
			len(n.Resource.RawConfig.Variables)+
			len(n.Resource.DependsOn))*2)
	copy(result, n.Resource.DependsOn)

	for _, v := range n.Resource.RawCount.Variables {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}
	for _, v := range n.Resource.RawConfig.Variables {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}
	for _, p := range n.Resource.Provisioners {
		for _, v := range p.ConnInfo.Variables {
			if vn := varNameForVar(v); vn != "" && vn != n.Resource.Id() {
				result = append(result, vn)
			}
		}
		for _, v := range p.RawConfig.Variables {
			if vn := varNameForVar(v); vn != "" && vn != n.Resource.Id() {
				result = append(result, vn)
			}
		}
	}

	return result
}

// VarWalk calls a callback for all the variables that this resource
// depends on.
func (n *GraphNodeConfigResource) VarWalk(fn func(config.InterpolatedVariable)) {
	for _, v := range n.Resource.RawCount.Variables {
		fn(v)
	}
	for _, v := range n.Resource.RawConfig.Variables {
		fn(v)
	}
	for _, p := range n.Resource.Provisioners {
		for _, v := range p.ConnInfo.Variables {
			fn(v)
		}
		for _, v := range p.RawConfig.Variables {
			fn(v)
		}
	}
}

func (n *GraphNodeConfigResource) Name() string {
	result := n.Resource.Id()
	if n.Destroy {
		result += " (destroy)"
	}
	return result
}
