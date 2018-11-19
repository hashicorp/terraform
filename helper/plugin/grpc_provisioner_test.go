package plugin

import proto "github.com/hashicorp/terraform/internal/tfplugin5"

var _ proto.ProvisionerServer = (*GRPCProvisionerServer)(nil)
