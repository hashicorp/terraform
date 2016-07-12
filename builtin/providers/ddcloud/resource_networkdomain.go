package ddcloud

import (
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strings"
	"time"
)

const (
	resourceKeyNetworkDomainName           = "name"
	resourceKeyNetworkDomainDescription    = "description"
	resourceKeyNetworkDomainPlan           = "plan"
	resourceKeyNetworkDomainDataCenter     = "datacenter"
	resourceKeyNetworkDomainNatIPv4Address = "nat_ipv4_address"
	resourceCreateTimeoutNetworkDomain     = 5 * time.Minute
	resourceDeleteTimeoutNetworkDomain     = 5 * time.Minute
)

func resourceNetworkDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkDomainCreate,
		Read:   resourceNetworkDomainRead,
		Update: resourceNetworkDomainUpdate,
		Delete: resourceNetworkDomainDelete,

		Schema: map[string]*schema.Schema{
			resourceKeyNetworkDomainName: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			resourceKeyNetworkDomainDescription: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			resourceKeyNetworkDomainPlan: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "ESSENTIALS",
				StateFunc: func(value interface{}) string {
					plan := value.(string)

					return strings.ToUpper(plan)
				},
			},
			resourceKeyNetworkDomainDataCenter: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			resourceKeyNetworkDomainNatIPv4Address: &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// Create a network domain resource.
func resourceNetworkDomainCreate(data *schema.ResourceData, provider interface{}) error {
	var name, description, plan, dataCenterID string

	name = data.Get(resourceKeyNetworkDomainName).(string)
	description = data.Get(resourceKeyNetworkDomainDescription).(string)
	plan = data.Get(resourceKeyNetworkDomainPlan).(string)
	dataCenterID = data.Get(resourceKeyNetworkDomainDataCenter).(string)

	log.Printf("Create network domain '%s' in data center '%s' (plan = '%s', description = '%s').", name, dataCenterID, plan, description)

	// TODO: Handle RESOURCE_BUSY response (retry?)
	apiClient := provider.(*providerState).Client()
	networkDomainID, err := apiClient.DeployNetworkDomain(name, description, plan, dataCenterID)
	if err != nil {
		return err
	}

	data.SetId(networkDomainID)

	log.Printf("Network domain '%s' is being provisioned...", networkDomainID)

	resource, err := apiClient.WaitForDeploy(compute.ResourceTypeNetworkDomain, networkDomainID, resourceCreateTimeoutVLAN)
	if err != nil {
		return err
	}

	// Capture additional properties that are only available after deployment.
	networkDomain := resource.(*compute.NetworkDomain)
	data.Set(resourceKeyNetworkDomainNatIPv4Address, networkDomain.NatIPv4Address)

	return nil
}

// Read a network domain resource.
func resourceNetworkDomainRead(data *schema.ResourceData, provider interface{}) error {
	var name, description, plan, dataCenterID string

	id := data.Id()
	name = data.Get(resourceKeyNetworkDomainName).(string)
	description = data.Get(resourceKeyNetworkDomainDescription).(string)
	plan = data.Get(resourceKeyNetworkDomainPlan).(string)
	dataCenterID = data.Get(resourceKeyNetworkDomainDataCenter).(string)

	log.Printf("Read network domain '%s' (Id = '%s') in data center '%s' (plan = '%s', description = '%s').", name, id, dataCenterID, plan, description)

	apiClient := provider.(*providerState).Client()

	networkDomain, err := apiClient.GetNetworkDomain(id)
	if err != nil {
		return err
	}

	if networkDomain != nil {
		data.Set(resourceKeyNetworkDomainName, networkDomain.Name)
		data.Set(resourceKeyNetworkDomainDescription, networkDomain.Description)
		data.Set(resourceKeyNetworkDomainPlan, networkDomain.Type)
		data.Set(resourceKeyNetworkDomainDataCenter, networkDomain.DatacenterID)
		data.Set(resourceKeyNetworkDomainNatIPv4Address, networkDomain.NatIPv4Address)
	} else {
		data.SetId("") // Mark resource as deleted.
	}

	return nil
}

// Update a network domain resource.
func resourceNetworkDomainUpdate(data *schema.ResourceData, provider interface{}) error {
	var (
		id, name, description, plan      string
		newName, newDescription, newPlan *string
	)

	id = data.Id()

	if data.HasChange(resourceKeyNetworkDomainName) {
		name = data.Get(resourceKeyNetworkDomainName).(string)
		newName = &name
	}

	if data.HasChange(resourceKeyNetworkDomainDescription) {
		description = data.Get(resourceKeyNetworkDomainDescription).(string)
		newDescription = &description
	}

	if data.HasChange(resourceKeyNetworkDomainPlan) {
		plan = data.Get(resourceKeyNetworkDomainPlan).(string)
		newPlan = &plan
	}

	log.Printf("Update network domain '%s' (Name = '%s', Description = '%s', Plan = '%s').", data.Id(), name, description, plan)

	apiClient := provider.(*providerState).Client()

	// TODO: Handle RESOURCE_BUSY response (retry?)
	return apiClient.EditNetworkDomain(id, newName, newDescription, newPlan)
}

// Delete a network domain resource.
func resourceNetworkDomainDelete(data *schema.ResourceData, provider interface{}) error {
	var err error

	networkDomainID := data.Id()
	name := data.Get(resourceKeyNetworkDomainName).(string)
	dataCenterID := data.Get(resourceKeyNetworkDomainDataCenter).(string)

	log.Printf("Delete network domain '%s' ('%s') in data center '%s'.", networkDomainID, name, dataCenterID)

	apiClient := provider.(*providerState).Client()

	// First, check if the network domain has any allocated public IP blocks.
	publicIPBlocks, err := apiClient.ListPublicIPBlocks(networkDomainID)
	if err != nil {
		return err
	}

	for _, block := range publicIPBlocks.Blocks {
		log.Printf("Removing public IP block '%s' (%s+%d) from network domain '%s'...", block.ID, block.BaseIP, block.Size, networkDomainID)

		err := apiClient.RemovePublicIPBlock(block.ID)
		if err != nil {
			return err
		}

		log.Printf("Successfully deleted public IP block '%s' from network domain '%s'.", block.ID, networkDomainID)
	}

	// TODO: Handle RESOURCE_BUSY response (retry?)
	err = apiClient.DeleteNetworkDomain(networkDomainID)
	if err != nil {
		return err
	}

	log.Printf("Network domain '%s' is being deleted...", networkDomainID)

	return apiClient.WaitForDelete(compute.ResourceTypeNetworkDomain, networkDomainID, resourceDeleteTimeoutServer)
}
