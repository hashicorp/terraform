package terraform

// NodeValidatableResource represents a resource that is used for validation
// only.
type NodeValidatableResource struct {
	*NodeAbstractResource
}

// GraphNodeEvalable
func (n *NodeValidatableResource) EvalTree() EvalNode {
	addr := n.NodeAbstractResource.Addr

	// Build the resource for eval
	resource := &Resource{
		Name:       addr.Name,
		Type:       addr.Type,
		CountIndex: addr.Index,
	}
	if resource.CountIndex < 0 {
		resource.CountIndex = 0
	}

	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var config *ResourceConfig
	var provider ResourceProvider

	seq := &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Name:   n.ProvidedBy()[0],
				Output: &provider,
			},
			&EvalInterpolate{
				Config:   n.Config.RawConfig.Copy(),
				Resource: resource,
				Output:   &config,
			},
			&EvalValidateResource{
				Provider:     &provider,
				Config:       &config,
				ResourceName: n.Config.Name,
				ResourceType: n.Config.Type,
				ResourceMode: n.Config.Mode,
			},
		},
	}

	// Validate all the provisioners
	for _, p := range n.Config.Provisioners {
		var provisioner ResourceProvisioner
		seq.Nodes = append(seq.Nodes, &EvalGetProvisioner{
			Name:   p.Type,
			Output: &provisioner,
		}, &EvalInterpolate{
			Config:   p.RawConfig.Copy(),
			Resource: resource,
			Output:   &config,
		}, &EvalValidateProvisioner{
			Provisioner: &provisioner,
			Config:      &config,
		})
	}

	return seq
}
