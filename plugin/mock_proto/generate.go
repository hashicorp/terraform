//go:generate mockgen -destination mock.go github.com/hashicorp/terraform/plugin/proto ProviderClient,ProvisionerClient,Provisioner_ProvisionResourceClient,Provisioner_ProvisionResourceServer

package mock_proto
