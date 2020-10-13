//go:generate go run github.com/golang/mock/mockgen -destination mock.go github.com/hashicorp/terraform/internal/tfplugin5 ProviderClient,ProvisionerClient,Provisioner_ProvisionResourceClient,Provisioner_ProvisionResourceServer

package mock_tfplugin5
