package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/Godeps/_workspace/src/github.com/Azure/go-autorest/autorest"
	"github.com/Azure/azure-sdk-for-go/Godeps/_workspace/src/github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/azure-sdk-for-go/arm/cdn"
	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/azure-sdk-for-go/arm/scheduler"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	mainStorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform/terraform"
	riviera "github.com/jen20/riviera/azure"
)

// ArmClient contains the handles to all the specific Azure Resource Manager
// resource classes' respective clients.
type ArmClient struct {
	rivieraClient *riviera.Client

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
	routeTablesClient            network.RouteTablesClient
	routesClient                 network.RoutesClient

	cdnProfilesClient  cdn.ProfilesClient
	cdnEndpointsClient cdn.EndpointsClient

	providers           resources.ProvidersClient
	resourceGroupClient resources.GroupsClient
	tagsClient          resources.TagsClient

	jobsClient            scheduler.JobsClient
	jobsCollectionsClient scheduler.JobCollectionsClient

	storageServiceClient storage.AccountsClient
	storageUsageClient   storage.UsageOperationsClient

	deploymentsClient resources.DeploymentsClient
}

func withRequestLogging() autorest.SendDecorator {
	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
			log.Printf("[DEBUG] Sending Azure RM Request %q to %q\n", r.Method, r.URL)
			resp, err := s.Do(r)
			if resp != nil {
				log.Printf("[DEBUG] Received Azure RM Request status code %s for %s\n", resp.Status, r.URL)
			} else {
				log.Printf("[DEBUG] Request to %s completed with no response", r.URL)
			}
			return resp, err
		})
	}
}

func withPollWatcher() autorest.SendDecorator {
	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
			fmt.Printf("[DEBUG] Sending Azure RM Request %q to %q\n", r.Method, r.URL)
			resp, err := s.Do(r)
			fmt.Printf("[DEBUG] Received Azure RM Request status code %s for %s\n", resp.Status, r.URL)
			if autorest.ResponseRequiresPolling(resp) {
				fmt.Printf("[DEBUG] Azure RM request will poll %s after %d seconds\n",
					autorest.GetPollingLocation(resp),
					int(autorest.GetPollingDelay(resp, time.Duration(0))/time.Second))
			}
			return resp, err
		})
	}
}

func setUserAgent(client *autorest.Client) {
	var version string
	if terraform.VersionPrerelease != "" {
		version = fmt.Sprintf("%s-%s", terraform.Version, terraform.VersionPrerelease)
	} else {
		version = terraform.Version
	}

	client.UserAgent = fmt.Sprintf("HashiCorp-Terraform-v%s", version)
}

// getArmClient is a helper method which returns a fully instantiated
// *ArmClient based on the Config's current settings.
func (c *Config) getArmClient() (*ArmClient, error) {
	// client declarations:
	client := ArmClient{}

	rivieraClient, err := riviera.NewClient(&riviera.AzureResourceManagerCredentials{
		ClientID:       c.ClientID,
		ClientSecret:   c.ClientSecret,
		TenantID:       c.TenantID,
		SubscriptionID: c.SubscriptionID,
	})
	if err != nil {
		return nil, fmt.Errorf("Error creating Riviera client: %s", err)
	}

	// validate that the credentials are correct using Riviera. Note that this must be
	// done _before_ using the Microsoft SDK, because Riviera handles errors. Using a
	// namespace registration instead of a simple OAuth token refresh guarantees that
	// service delegation is correct. This has the effect of registering Microsoft.Compute
	// which is neccessary anyway.
	if err := registerProviderWithSubscription("Microsoft.Compute", rivieraClient); err != nil {
		return nil, err
	}
	client.rivieraClient = rivieraClient

	spt, err := azure.NewServicePrincipalToken(c.ClientID, c.ClientSecret, c.TenantID, azure.AzureResourceManagerScope)
	if err != nil {
		return nil, err
	}

	// NOTE: these declarations should be left separate for clarity should the
	// clients be wished to be configured with custom Responders/PollingModess etc...
	asc := compute.NewAvailabilitySetsClient(c.SubscriptionID)
	setUserAgent(&asc.Client)
	asc.Authorizer = spt
	asc.Sender = autorest.CreateSender(withRequestLogging())
	client.availSetClient = asc

	uoc := compute.NewUsageOperationsClient(c.SubscriptionID)
	setUserAgent(&uoc.Client)
	uoc.Authorizer = spt
	uoc.Sender = autorest.CreateSender(withRequestLogging())
	client.usageOpsClient = uoc

	vmeic := compute.NewVirtualMachineExtensionImagesClient(c.SubscriptionID)
	setUserAgent(&vmeic.Client)
	vmeic.Authorizer = spt
	vmeic.Sender = autorest.CreateSender(withRequestLogging())
	client.vmExtensionImageClient = vmeic

	vmec := compute.NewVirtualMachineExtensionsClient(c.SubscriptionID)
	setUserAgent(&vmec.Client)
	vmec.Authorizer = spt
	vmec.Sender = autorest.CreateSender(withRequestLogging())
	client.vmExtensionClient = vmec

	vmic := compute.NewVirtualMachineImagesClient(c.SubscriptionID)
	setUserAgent(&vmic.Client)
	vmic.Authorizer = spt
	vmic.Sender = autorest.CreateSender(withRequestLogging())
	client.vmImageClient = vmic

	vmc := compute.NewVirtualMachinesClient(c.SubscriptionID)
	setUserAgent(&vmc.Client)
	vmc.Authorizer = spt
	vmc.Sender = autorest.CreateSender(withRequestLogging())
	client.vmClient = vmc

	agc := network.NewApplicationGatewaysClient(c.SubscriptionID)
	setUserAgent(&agc.Client)
	agc.Authorizer = spt
	agc.Sender = autorest.CreateSender(withRequestLogging())
	client.appGatewayClient = agc

	ifc := network.NewInterfacesClient(c.SubscriptionID)
	setUserAgent(&ifc.Client)
	ifc.Authorizer = spt
	ifc.Sender = autorest.CreateSender(withRequestLogging())
	client.ifaceClient = ifc

	lbc := network.NewLoadBalancersClient(c.SubscriptionID)
	setUserAgent(&lbc.Client)
	lbc.Authorizer = spt
	lbc.Sender = autorest.CreateSender(withRequestLogging())
	client.loadBalancerClient = lbc

	lgc := network.NewLocalNetworkGatewaysClient(c.SubscriptionID)
	setUserAgent(&lgc.Client)
	lgc.Authorizer = spt
	lgc.Sender = autorest.CreateSender(withRequestLogging())
	client.localNetConnClient = lgc

	pipc := network.NewPublicIPAddressesClient(c.SubscriptionID)
	setUserAgent(&pipc.Client)
	pipc.Authorizer = spt
	pipc.Sender = autorest.CreateSender(withRequestLogging())
	client.publicIPClient = pipc

	sgc := network.NewSecurityGroupsClient(c.SubscriptionID)
	setUserAgent(&sgc.Client)
	sgc.Authorizer = spt
	sgc.Sender = autorest.CreateSender(withRequestLogging())
	client.secGroupClient = sgc

	src := network.NewSecurityRulesClient(c.SubscriptionID)
	setUserAgent(&src.Client)
	src.Authorizer = spt
	src.Sender = autorest.CreateSender(withRequestLogging())
	client.secRuleClient = src

	snc := network.NewSubnetsClient(c.SubscriptionID)
	setUserAgent(&snc.Client)
	snc.Authorizer = spt
	snc.Sender = autorest.CreateSender(withRequestLogging())
	client.subnetClient = snc

	vgcc := network.NewVirtualNetworkGatewayConnectionsClient(c.SubscriptionID)
	setUserAgent(&vgcc.Client)
	vgcc.Authorizer = spt
	vgcc.Sender = autorest.CreateSender(withRequestLogging())
	client.vnetGatewayConnectionsClient = vgcc

	vgc := network.NewVirtualNetworkGatewaysClient(c.SubscriptionID)
	setUserAgent(&vgc.Client)
	vgc.Authorizer = spt
	vgc.Sender = autorest.CreateSender(withRequestLogging())
	client.vnetGatewayClient = vgc

	vnc := network.NewVirtualNetworksClient(c.SubscriptionID)
	setUserAgent(&vnc.Client)
	vnc.Authorizer = spt
	vnc.Sender = autorest.CreateSender(withRequestLogging())
	client.vnetClient = vnc

	rtc := network.NewRouteTablesClient(c.SubscriptionID)
	setUserAgent(&rtc.Client)
	rtc.Authorizer = spt
	rtc.Sender = autorest.CreateSender(withRequestLogging())
	client.routeTablesClient = rtc

	rc := network.NewRoutesClient(c.SubscriptionID)
	setUserAgent(&rc.Client)
	rc.Authorizer = spt
	rc.Sender = autorest.CreateSender(withRequestLogging())
	client.routesClient = rc

	rgc := resources.NewGroupsClient(c.SubscriptionID)
	setUserAgent(&rgc.Client)
	rgc.Authorizer = spt
	rgc.Sender = autorest.CreateSender(withRequestLogging())
	client.resourceGroupClient = rgc

	pc := resources.NewProvidersClient(c.SubscriptionID)
	setUserAgent(&pc.Client)
	pc.Authorizer = spt
	pc.Sender = autorest.CreateSender(withRequestLogging())
	client.providers = pc

	tc := resources.NewTagsClient(c.SubscriptionID)
	setUserAgent(&tc.Client)
	tc.Authorizer = spt
	tc.Sender = autorest.CreateSender(withRequestLogging())
	client.tagsClient = tc

	jc := scheduler.NewJobsClient(c.SubscriptionID)
	setUserAgent(&jc.Client)
	jc.Authorizer = spt
	jc.Sender = autorest.CreateSender(withRequestLogging())
	client.jobsClient = jc

	jcc := scheduler.NewJobCollectionsClient(c.SubscriptionID)
	setUserAgent(&jcc.Client)
	jcc.Authorizer = spt
	jcc.Sender = autorest.CreateSender(withRequestLogging())
	client.jobsCollectionsClient = jcc

	ssc := storage.NewAccountsClient(c.SubscriptionID)
	setUserAgent(&ssc.Client)
	ssc.Authorizer = spt
	ssc.Sender = autorest.CreateSender(withRequestLogging(), withPollWatcher())
	client.storageServiceClient = ssc

	suc := storage.NewUsageOperationsClient(c.SubscriptionID)
	setUserAgent(&suc.Client)
	suc.Authorizer = spt
	suc.Sender = autorest.CreateSender(withRequestLogging())
	client.storageUsageClient = suc

	cpc := cdn.NewProfilesClient(c.SubscriptionID)
	setUserAgent(&cpc.Client)
	cpc.Authorizer = spt
	cpc.Sender = autorest.CreateSender(withRequestLogging())
	client.cdnProfilesClient = cpc

	cec := cdn.NewEndpointsClient(c.SubscriptionID)
	setUserAgent(&cec.Client)
	cec.Authorizer = spt
	cec.Sender = autorest.CreateSender(withRequestLogging())
	client.cdnEndpointsClient = cec

	dc := resources.NewDeploymentsClient(c.SubscriptionID)
	setUserAgent(&dc.Client)
	dc.Authorizer = spt
	dc.Sender = autorest.CreateSender(withRequestLogging())
	client.deploymentsClient = dc

	return &client, nil
}

func (armClient *ArmClient) getKeyForStorageAccount(resourceGroupName, storageAccountName string) (string, error) {
	keys, err := armClient.storageServiceClient.ListKeys(resourceGroupName, storageAccountName)
	if err != nil {
		return "", fmt.Errorf("Error retrieving keys for storage account %q: %s", storageAccountName, err)
	}

	if keys.Key1 == nil {
		return "", fmt.Errorf("Nil key returned for storage account %q", storageAccountName)
	}

	return *keys.Key1, nil
}

func (armClient *ArmClient) getBlobStorageClientForStorageAccount(resourceGroupName, storageAccountName string) (*mainStorage.BlobStorageClient, error) {
	key, err := armClient.getKeyForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return nil, err
	}

	storageClient, err := mainStorage.NewBasicClient(storageAccountName, key)
	if err != nil {
		return nil, fmt.Errorf("Error creating storage client for storage account %q: %s", storageAccountName, err)
	}

	blobClient := storageClient.GetBlobService()
	return &blobClient, nil
}
func (armClient *ArmClient) getQueueServiceClientForStorageAccount(resourceGroupName, storageAccountName string) (*mainStorage.QueueServiceClient, error) {
	key, err := armClient.getKeyForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return nil, err
	}

	storageClient, err := mainStorage.NewBasicClient(storageAccountName, key)
	if err != nil {
		return nil, fmt.Errorf("Error creating storage client for storage account %q: %s", storageAccountName, err)
	}

	queueClient := storageClient.GetQueueService()
	return &queueClient, nil
}
