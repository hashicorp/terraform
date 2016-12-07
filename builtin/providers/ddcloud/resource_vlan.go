package ddcloud

import (
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"time"
)

const (
	resourceKeyVLANNetworkDomainID = "networkdomain"
	resourceKeyVLANName            = "name"
	resourceKeyVLANDescription     = "description"
	resourceKeyVLANIPv4BaseAddress = "ipv4_base_address"
	resourceKeyVLANIPv4PrefixSize  = "ipv4_prefix_size"
	resourceKeyVLANIPv6BaseAddress = "ipv6_base_address"
	resourceKeyVLANIPv6PrefixSize  = "ipv6_prefix_size"
	resourceCreateTimeoutVLAN      = 5 * time.Minute
	resourceDeleteTimeoutVLAN      = 5 * time.Minute
)

func resourceVLAN() *schema.Resource {
	return &schema.Resource{
		Create: resourceVLANCreate,
		Read:   resourceVLANRead,
		Update: resourceVLANUpdate,
		Delete: resourceVLANDelete,

		Schema: map[string]*schema.Schema{
			resourceKeyVLANNetworkDomainID: &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The Id of the network domain in which the VLAN is deployed.",
			},
			resourceKeyVLANName: &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The VLAN display name.",
			},
			resourceKeyVLANDescription: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The VLAN description.",
			},
			resourceKeyVLANIPv4BaseAddress: &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The VLAN's private IPv4 base address.",
			},
			resourceKeyVLANIPv4PrefixSize: &schema.Schema{
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The VLAN's private IPv4 prefix length.",
			},
			resourceKeyVLANIPv6BaseAddress: &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The VLAN's IPv6 base address.",
			},
			resourceKeyVLANIPv6PrefixSize: &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The VLAN's IPv6 prefix length.",
			},
		},
	}
}

// Create a VLAN resource.
func resourceVLANCreate(data *schema.ResourceData, provider interface{}) error {
	var (
		networkDomainID, name, description, ipv4BaseAddress string
		ipv4PrefixSize                                      int
	)

	networkDomainID = data.Get(resourceKeyVLANNetworkDomainID).(string)
	name = data.Get(resourceKeyVLANName).(string)
	description = data.Get(resourceKeyVLANDescription).(string)
	ipv4BaseAddress = data.Get(resourceKeyVLANIPv4BaseAddress).(string)
	ipv4PrefixSize = data.Get(resourceKeyVLANIPv4PrefixSize).(int)

	log.Printf("Create VLAN '%s' ('%s') in network domain '%s' (IPv4 network = '%s/%d').", name, description, networkDomainID, ipv4BaseAddress, ipv4PrefixSize)

	providerState := provider.(*providerState)
	apiClient := providerState.Client()

	domainLock := providerState.GetDomainLock(networkDomainID, "resourceVLANCreate(name = '%s')", name)
	domainLock.Lock()
	defer domainLock.Unlock()

	// TODO: Handle RESOURCE_BUSY response (retry?)
	vlanID, err := apiClient.DeployVLAN(networkDomainID, name, description, ipv4BaseAddress, ipv4PrefixSize)
	if err != nil {
		return err
	}

	data.SetId(vlanID)

	log.Printf("VLAN '%s' is being provisioned...", vlanID)

	deployedResource, err := apiClient.WaitForDeploy(compute.ResourceTypeVLAN, vlanID, resourceCreateTimeoutVLAN)
	if err != nil {
		return err
	}

	vlan := deployedResource.(*compute.VLAN)
	data.Set(resourceKeyVLANIPv6BaseAddress, vlan.IPv6Range.BaseAddress)
	data.Set(resourceKeyVLANIPv6PrefixSize, vlan.IPv6Range.PrefixSize)

	return nil
}

// Read a VLAN resource.
func resourceVLANRead(data *schema.ResourceData, provider interface{}) error {
	id := data.Id()
	networkDomainID := data.Get(resourceKeyVLANNetworkDomainID).(string)
	name := data.Get(resourceKeyVLANName).(string)
	description := data.Get(resourceKeyVLANDescription).(string)
	ipv4BaseAddress := data.Get(resourceKeyVLANIPv4BaseAddress).(string)
	ipv4PrefixSize := data.Get(resourceKeyVLANIPv4PrefixSize).(int)
	ipv6BaseAddress := data.Get(resourceKeyVLANIPv6BaseAddress).(string)
	ipv6PrefixSize := data.Get(resourceKeyVLANIPv6PrefixSize).(int)

	log.Printf("Read VLAN '%s' (Name = '%s', description = '%s') in network domain '%s' (IPv4 network = '%s/%d', IPv6 network = '%s/%d').", id, name, description, networkDomainID, ipv4BaseAddress, ipv4PrefixSize, ipv6BaseAddress, ipv6PrefixSize)

	apiClient := provider.(*providerState).Client()

	vlan, err := apiClient.GetVLAN(id)
	if err != nil {
		return err
	}

	if vlan != nil {
		data.Set(resourceKeyVLANName, vlan.Name)
		data.Set(resourceKeyVLANDescription, vlan.Description)
		data.Set(resourceKeyVLANIPv4BaseAddress, vlan.IPv4Range.BaseAddress)
		data.Set(resourceKeyVLANIPv4PrefixSize, vlan.IPv4Range.PrefixSize)
		data.Set(resourceKeyVLANIPv6BaseAddress, vlan.IPv6Range.BaseAddress)
		data.Set(resourceKeyVLANIPv6PrefixSize, vlan.IPv6Range.PrefixSize)
	} else {
		data.SetId("") // Mark resource as deleted.
	}

	return nil
}

// Update a VLAN resource.
func resourceVLANUpdate(data *schema.ResourceData, provider interface{}) error {
	var (
		id, ipv4BaseAddress, name, description string
		newName, newDescription                *string
		ipv4PrefixSize                         int
	)

	id = data.Id()
	networkDomainID := data.Get(resourceKeyVLANNetworkDomainID).(string)

	name = data.Get(resourceKeyVLANName).(string)
	if data.HasChange(resourceKeyVLANName) {
		newName = &name
	}

	if data.HasChange(resourceKeyVLANDescription) {
		description = data.Get(resourceKeyVLANDescription).(string)
		newDescription = &description
	}

	log.Printf("Update VLAN '%s' (name = '%s', description = '%s', IPv4 network = '%s/%d').", id, name, description, ipv4BaseAddress, ipv4PrefixSize)

	providerState := provider.(*providerState)
	apiClient := providerState.Client()

	domainLock := providerState.GetDomainLock(networkDomainID, "resourceVLANUpdate(id = '%s', name = '%s')", id, name)
	domainLock.Lock()
	defer domainLock.Unlock()

	if newName == nil && newDescription == nil {
		return nil
	}

	// TODO: Handle RESOURCE_BUSY response (retry?)
	return apiClient.EditVLAN(id, newName, newDescription)
}

// Delete a VLAN resource.
func resourceVLANDelete(data *schema.ResourceData, provider interface{}) error {
	id := data.Id()
	name := data.Get(resourceKeyVLANName).(string)
	networkDomainID := data.Get(resourceKeyVLANNetworkDomainID).(string)

	log.Printf("Delete VLAN '%s' ('%s') in network domain '%s'.", id, name, networkDomainID)

	providerState := provider.(*providerState)
	apiClient := providerState.Client()

	domainLock := providerState.GetDomainLock(networkDomainID, "resourceVLANDelete(id = '%s', name = '%s')", id, name)
	domainLock.Lock()
	defer domainLock.Unlock()

	err := apiClient.DeleteVLAN(id)
	if err != nil {
		return err
	}

	log.Printf("VLAN '%s' is being deleted...", id)

	// TODO: Handle RESOURCE_BUSY response (retry?)
	return apiClient.WaitForDelete(compute.ResourceTypeVLAN, id, resourceDeleteTimeoutServer)
}
