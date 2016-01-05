package azurerm

import (
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/Godeps/_workspace/src/github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources"
	"github.com/Azure/azure-sdk-for-go/arm/scheduler"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/Azure/go-autorest/autorest"
)

// ArmClient contains the handles to all the specific Azure Resource Manager
// resource classes' respective clients.
type ArmClient struct {
	availSetClient         compute.AvailabilitySetsClient
	usageOpsClient         compute.UsageOperationsClient
	vmExtensionImageClient compute.VirtualMachineExtensionImagesClient
	vmExtensionClient      compute.VirtualMachineExtensionsClient
	vmImageClient          compute.VirtualMachineImagesClient
	vmClient               compute.VirtualMachinesClient

	appGatewayClient             network.ApplicationGatewaysClient
	ifaceClient                  network.InterfacesClient
	loadBalancerClient           network.LoadBalancersClient
	localNetConnClient           network.LocalNetworkGatewaysClient
	publicIPClient               network.PublicIPAddressesClient
	secGroupClient               network.SecurityGroupsClient
	secRuleClient                network.SecurityRulesClient
	subnetClient                 network.SubnetsClient
	netUsageClient               network.UsagesClient
	vnetGatewayConnectionsClient network.VirtualNetworkGatewayConnectionsClient
	vnetGatewayClient            network.VirtualNetworkGatewaysClient
	vnetClient                   network.VirtualNetworksClient

	providers           resources.ProvidersClient
	resourceGroupClient resources.GroupsClient
	tagsClient          resources.TagsClient

	jobsClient            scheduler.JobsClient
	jobsCollectionsClient scheduler.JobCollectionsClient

	storageServiceClient storage.AccountsClient
	storageUsageClient   storage.UsageOperationsClient
}

func withRequestLogging() autorest.SendDecorator {
	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
			log.Printf("[DEBUG] Sending Azure RM Request %s to %s\n", r.Method, r.URL)
			resp, err := s.Do(r)
			log.Printf("[DEBUG] Received Azure RM Request status code %s for %s\n", resp.Status, r.URL)
			return resp, err
		})
	}
}

// getArmClient is a helper method which returns a fully instantiated
// *ArmClient based on the Config's current settings.
func (c *Config) getArmClient() (*ArmClient, error) {
	spt, err := azure.NewServicePrincipalToken(c.ClientID, c.ClientSecret, c.TenantID, azure.AzureResourceManagerScope)
	if err != nil {
		return nil, err
	}

	// client declarations:
	client := ArmClient{}

	// NOTE: these declarations should be left separate for clarity should the
	// clients be wished to be configured with custom Responders/PollingModess etc...
	asc := compute.NewAvailabilitySetsClient(c.SubscriptionID)
	asc.Authorizer = spt
	asc.Sender = autorest.CreateSender(withRequestLogging())
	client.availSetClient = asc

	uoc := compute.NewUsageOperationsClient(c.SubscriptionID)
	uoc.Authorizer = spt
	uoc.Sender = autorest.CreateSender(withRequestLogging())
	client.usageOpsClient = uoc

	vmeic := compute.NewVirtualMachineExtensionImagesClient(c.SubscriptionID)
	vmeic.Authorizer = spt
	vmeic.Sender = autorest.CreateSender(withRequestLogging())
	client.vmExtensionImageClient = vmeic

	vmec := compute.NewVirtualMachineExtensionsClient(c.SubscriptionID)
	vmec.Authorizer = spt
	vmec.Sender = autorest.CreateSender(withRequestLogging())
	client.vmExtensionClient = vmec

	vmic := compute.NewVirtualMachineImagesClient(c.SubscriptionID)
	vmic.Authorizer = spt
	vmic.Sender = autorest.CreateSender(withRequestLogging())
	client.vmImageClient = vmic

	vmc := compute.NewVirtualMachinesClient(c.SubscriptionID)
	vmc.Authorizer = spt
	vmc.Sender = autorest.CreateSender(withRequestLogging())
	client.vmClient = vmc

	agc := network.NewApplicationGatewaysClient(c.SubscriptionID)
	agc.Authorizer = spt
	agc.Sender = autorest.CreateSender(withRequestLogging())
	client.appGatewayClient = agc

	ifc := network.NewInterfacesClient(c.SubscriptionID)
	ifc.Authorizer = spt
	ifc.Sender = autorest.CreateSender(withRequestLogging())
	client.ifaceClient = ifc

	lbc := network.NewLoadBalancersClient(c.SubscriptionID)
	lbc.Authorizer = spt
	lbc.Sender = autorest.CreateSender(withRequestLogging())
	client.loadBalancerClient = lbc

	lgc := network.NewLocalNetworkGatewaysClient(c.SubscriptionID)
	lgc.Authorizer = spt
	lgc.Sender = autorest.CreateSender(withRequestLogging())
	client.localNetConnClient = lgc

	pipc := network.NewPublicIPAddressesClient(c.SubscriptionID)
	pipc.Authorizer = spt
	pipc.Sender = autorest.CreateSender(withRequestLogging())
	client.publicIPClient = pipc

	sgc := network.NewSecurityGroupsClient(c.SubscriptionID)
	sgc.Authorizer = spt
	sgc.Sender = autorest.CreateSender(withRequestLogging())
	client.secGroupClient = sgc

	src := network.NewSecurityRulesClient(c.SubscriptionID)
	src.Authorizer = spt
	src.Sender = autorest.CreateSender(withRequestLogging())
	client.secRuleClient = src

	snc := network.NewSubnetsClient(c.SubscriptionID)
	snc.Authorizer = spt
	snc.Sender = autorest.CreateSender(withRequestLogging())
	client.subnetClient = snc

	vgcc := network.NewVirtualNetworkGatewayConnectionsClient(c.SubscriptionID)
	vgcc.Authorizer = spt
	vgcc.Sender = autorest.CreateSender(withRequestLogging())
	client.vnetGatewayConnectionsClient = vgcc

	vgc := network.NewVirtualNetworkGatewaysClient(c.SubscriptionID)
	vgc.Authorizer = spt
	vgc.Sender = autorest.CreateSender(withRequestLogging())
	client.vnetGatewayClient = vgc

	vnc := network.NewVirtualNetworksClient(c.SubscriptionID)
	vnc.Authorizer = spt
	vnc.Sender = autorest.CreateSender(withRequestLogging())
	client.vnetClient = vnc

	rgc := resources.NewGroupsClient(c.SubscriptionID)
	rgc.Authorizer = spt
	rgc.Sender = autorest.CreateSender(withRequestLogging())
	client.resourceGroupClient = rgc

	pc := resources.NewProvidersClient(c.SubscriptionID)
	pc.Authorizer = spt
	pc.Sender = autorest.CreateSender(withRequestLogging())
	client.providers = pc

	tc := resources.NewTagsClient(c.SubscriptionID)
	tc.Authorizer = spt
	tc.Sender = autorest.CreateSender(withRequestLogging())
	client.tagsClient = tc

	jc := scheduler.NewJobsClient(c.SubscriptionID)
	jc.Authorizer = spt
	jc.Sender = autorest.CreateSender(withRequestLogging())
	client.jobsClient = jc

	jcc := scheduler.NewJobCollectionsClient(c.SubscriptionID)
	jcc.Authorizer = spt
	jcc.Sender = autorest.CreateSender(withRequestLogging())
	client.jobsCollectionsClient = jcc

	ssc := storage.NewAccountsClient(c.SubscriptionID)
	ssc.Authorizer = spt
	ssc.Sender = autorest.CreateSender(withRequestLogging())
	client.storageServiceClient = ssc

	suc := storage.NewUsageOperationsClient(c.SubscriptionID)
	suc.Authorizer = spt
	suc.Sender = autorest.CreateSender(withRequestLogging())
	client.storageUsageClient = suc

	return &client, nil
}
