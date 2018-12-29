package plugin

import "github.com/hashicorp/terraform/plugin/proto"

var _ proto.ProvisionerServer = (*GRPCProvisionerServer)(nil)
