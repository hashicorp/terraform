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

	apiClient := provider.(*compute.Client)

	// TODO: Handle RESOURCE_BUSY response (retry?)
	vlanID, err := apiClient.DeployVLAN(networkDomainID, name, description, ipv4BaseAddress, ipv4PrefixSize)
	if err != nil {
		return err
	}

	data.SetId(vlanID)

	log.Printf("VLAN '%s' is being provisioned...", vlanID)

	_, err = apiClient.WaitForDeploy(compute.ResourceTypeVLAN, vlanID, resourceCreateTimeoutVLAN)

	return err
}

// Read a VLAN resource.
func resourceVLANRead(data *schema.ResourceData, provider interface{}) error {
	var (
		id, networkDomainID, name, description, ipv4BaseAddress string
		ipv4PrefixSize                                          int
	)

	id = data.Id()
	networkDomainID = data.Get(resourceKeyVLANNetworkDomainID).(string)
	name = data.Get(resourceKeyVLANName).(string)
	description = data.Get(resourceKeyVLANDescription).(string)
	ipv4BaseAddress = data.Get(resourceKeyVLANIPv4BaseAddress).(string)
	ipv4PrefixSize = data.Get(resourceKeyVLANIPv4PrefixSize).(int)

	log.Printf("Read VLAN '%s' (Name = '%s', description = '%s') in network domain '%s' (IPv4 network = '%s/%d').", id, name, description, networkDomainID, ipv4BaseAddress, ipv4PrefixSize)

	apiClient := provider.(*compute.Client)

	vlan, err := apiClient.GetVLAN(id)
	if err != nil {
		return err
	}

	if vlan != nil {
		data.Set(resourceKeyVLANName, vlan.Name)
		data.Set(resourceKeyVLANDescription, vlan.Description)
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

	if data.HasChange(resourceKeyVLANName) {
		name = data.Get(resourceKeyVLANName).(string)
		newName = &name
	}

	if data.HasChange(resourceKeyVLANDescription) {
		description = data.Get(resourceKeyVLANDescription).(string)
		newDescription = &description
	}

	log.Printf("Update VLAN '%s' (name = '%s', description = '%s', IPv4 network = '%s/%d').", id, name, description, ipv4BaseAddress, ipv4PrefixSize)

	apiClient := provider.(*compute.Client)

	// TODO: Handle RESOURCE_BUSY response (retry?)
	if newName != nil || newDescription != nil {
		err := apiClient.EditVLAN(id, newName, newDescription)
		if err != nil {
			return err
		}
	}

	return nil
}

// Delete a VLAN resource.
func resourceVLANDelete(data *schema.ResourceData, provider interface{}) error {
	id := data.Id()
	name := data.Get(resourceKeyVLANName).(string)
	networkDomainID := data.Get(resourceKeyVLANNetworkDomainID).(string)

	log.Printf("Delete VLAN '%s' ('%s') in network domain '%s'.", id, name, networkDomainID)

	apiClient := provider.(*compute.Client)
	err := apiClient.DeleteVLAN(id)
	if err != nil {
		return err
	}

	log.Printf("VLAN '%s' is being deleted...", id)

	// TODO: Handle RESOURCE_BUSY response (retry?)
	return apiClient.WaitForDelete(compute.ResourceTypeVLAN, id, resourceDeleteTimeoutServer)
}
