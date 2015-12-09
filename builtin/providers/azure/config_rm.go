package azure

import (
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/Godeps/_workspace/src/github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources"
	"github.com/Azure/azure-sdk-for-go/arm/scheduler"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/hashicorp/terraform/helper/pathorcontents"
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
	publicIpClient               network.PublicIPAddressesClient
	secGroupClient               network.SecurityGroupsClient
	secRuleClient                network.SecurityRulesClient
	subnetClient                 network.SubnetsClient
	netUsageClient               network.UsagesClient
	vnetGatewayConnectionsClient network.VirtualNetworkGatewayConnectionsClient
	vnetGatewayClient            network.VirtualNetworkGatewaysClient
	vnetClient                   network.VirtualNetworksClient

	resourceGroupClient resources.GroupsClient
	tagsClient          resources.TagsClient

	jobsClient            scheduler.JobsClient
	jobsCollectionsClient scheduler.JobCollectionsClient

	storageServiceClient storage.AccountsClient
	storageUsageClient   storage.UsageOperationsClient
}

// getArmClient is a helper method which returns a fully instantiated
// *ArmClient based on the Config's current settings.
func (c *Config) getArmClient() (*ArmClient, error) {
	// first; check that all the necessary credentials were provided:
	if !c._armCredentialsProvided() {
		return nil, fmt.Errorf("Not all ARM-required fields have been provided.")
	}

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
	client.availSetClient = asc

	uoc := compute.NewUsageOperationsClient(c.SubscriptionID)
	uoc.Authorizer = spt
	client.usageOpsClient = uoc

	vmeic := compute.NewVirtualMachineExtensionImagesClient(c.SubscriptionID)
	vmeic.Authorizer = spt
	client.vmExtensionImageClient = vmeic

	vmec := compute.NewVirtualMachineExtensionsClient(c.SubscriptionID)
	vmec.Authorizer = spt
	client.vmExtensionClient = vmec

	vmic := compute.NewVirtualMachineImagesClient(c.SubscriptionID)
	vmic.Authorizer = spt
	client.vmImageClient = vmic

	vmc := compute.NewVirtualMachinesClient(c.SubscriptionID)
	vmc.Authorizer = spt
	client.vmClient = vmc

	agc := network.NewApplicationGatewaysClient(c.SubscriptionID)
	agc.Authorizer = spt
	client.appGatewayClient = agc

	ifc := network.NewInterfacesClient(c.SubscriptionID)
	ifc.Authorizer = spt
	client.ifaceClient = ifc

	lbc := network.NewLoadBalancersClient(c.SubscriptionID)
	lbc.Authorizer = spt
	client.loadBalancerClient = lbc

	lgc := network.NewLocalNetworkGatewaysClient(c.SubscriptionID)
	lgc.Authorizer = spt
	client.localNetConnClient = lgc

	pipc := network.NewPublicIPAddressesClient(c.SubscriptionID)
	pipc.Authorizer = spt
	client.publicIpClient = pipc

	sgc := network.NewSecurityGroupsClient(c.SubscriptionID)
	sgc.Authorizer = spt
	client.secGroupClient = sgc

	src := network.NewSecurityRulesClient(c.SubscriptionID)
	src.Authorizer = spt
	client.secRuleClient = src

	snc := network.NewSubnetsClient(c.SubscriptionID)
	snc.Authorizer = spt
	client.subnetClient = snc

	vgcc := network.NewVirtualNetworkGatewayConnectionsClient(c.SubscriptionID)
	vgcc.Authorizer = spt
	client.vnetGatewayConnectionsClient = vgcc

	vgc := network.NewVirtualNetworkGatewaysClient(c.SubscriptionID)
	vgc.Authorizer = spt
	client.vnetGatewayClient = vgc

	vnc := network.NewVirtualNetworksClient(c.SubscriptionID)
	vnc.Authorizer = spt
	client.vnetClient = vnc

	rgc := resources.NewGroupsClient(c.SubscriptionID)
	rgc.Authorizer = rgc
	client.resourceGroupClient = rgc

	tc := resources.NewTagsClient(c.SubscriptionID)
	tc.Authorizer = spt
	client.tagsClient = tc

	jc := scheduler.NewJobsClient(c.SubscriptionID)
	jc.Authorizer = spt
	client.jobsClient = jc

	jcc := scheduler.NewJobCollectionsClient(c.SubscriptionID)
	jcc.Authorizer = spt
	client.jobsCollectionsClient = jcc

	ssc := storage.NewAccountsClient(c.SubscriptionID)
	ssc.Authorizer = spt
	client.storageServiceClient = ssc

	suc := storage.NewUsageOperationsClient(c.SubscriptionID)
	suc.Authorizer = spt
	client.storageUsageClient = suc

	return &client, nil
}

// armCredentialsProvided is a helper method which indicates whether or not the
// credentials required for authenticating against the ARM APIs were provided.
func (c *Config) armCredentialsProvided() bool {
	return c.ArmConfig != "" || c._armCredentialsProvided()
}
func (c *Config) _armCredentialsProvided() bool {
	return !(c.SubscriptionID == "" || c.ClientID == "" || c.ClientSecret == "" || c.TenantID == "")
}

// readArmSettings is a helper method which; given the contents of the ARM
// credentials file, loads all the data into the Config.
func (c *Config) readArmSettings(contents string) error {
	data := &armConfigData{}
	err := json.Unmarshal([]byte(contents), data)

	c.SubscriptionID = data.SubscriptionID
	c.ClientID = data.ClientID
	c.ClientSecret = data.ClientSecret
	c.TenantID = data.TenantID

	return err
}

// configFileContentsWarning represents the warning message returned when the
// path to the 'arm_config_file' is provided instead of its sourced contents.
var configFileContentsWarning = `
The path to the 'arm_config_file' was provided instead of its contents.
Support for accepting filepaths instead of their contents will be removed
in the near future. Do please consider switching over to using
'${file("/path/to/config.arm")}' instead.
`[1:]

// validateArmConfigFile is a helper function which verifies that
// the provided ARM configuration file is valid.
func validateArmConfigFile(v interface{}, _ string) (ws []string, es []error) {
	value := v.(string)
	if value == "" {
		return nil, nil
	}

	pathOrContents, wasPath, err := pathorcontents.Read(v.(string))
	if err != nil {
		es = append(es, fmt.Errorf("Error reading 'arm_config_file': %s", err))
	}

	if wasPath {
		ws = append(ws, configFileContentsWarning)
	}

	data := armConfigData{}
	err = json.Unmarshal([]byte(pathOrContents), &data)
	if err != nil {
		es = append(es, fmt.Errorf("Error unmarshalling the provided 'arm_config_file': %s", err))
	}

	return
}

// armConfigData is a private struct which represents the expected layout of
// an ARM configuration file. It is used for unmarshalling purposes.
type armConfigData struct {
	ClientID       string `json:"clientID"`
	ClientSecret   string `json:"clientSecret"`
	SubscriptionID string `json:"subscriptionID"`
	TenantID       string `json:"tenantID"`
}
