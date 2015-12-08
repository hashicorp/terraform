package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmVirtualNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmVirtualNetworkCreate,
		Read:   resourceArmVirtualNetworkRead,
		Update: resourceArmVirtualNetworkUpdate,
		Delete: resourceArmVirtualNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"address_space": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"dns_servers_names": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"subnet": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"address_prefix": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"security_group": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceAzureSubnetHash,
			},

			"location": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

// resourceArmVirtualNetworkCreate creates the specified ARM virtual network.
func resourceArmVirtualNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	vnetClient := client.vnetClient

	log.Printf("[INFO] preparing arguments for Azure ARM virtual network creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)

	vnet := network.VirtualNetwork{
		Name:       &name,
		Location:   &location,
		Properties: getVirtualNetworkProperties(d),
	}

	log.Printf("[INFO] Sending virtual network create request to ARM.")
	_, err := vnetClient.CreateOrUpdate(resGroup, name, vnet)
	if err != nil {
		return err
	}

	// if res.Response.StatusCode != http.StatusAccepted {
	// 	return fmt.Errorf("Creation request was denies: code: %d", res.Response.StatusCode)
	// }

	d.SetId(name)
	d.Set("resGroup", resGroup)

	// Wait for the resource group to become available
	// TODO(jen20): Is there any need for this?
	log.Printf("[DEBUG] Waiting for Virtual Network (%s) to become available", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  "Succeeded",
		Refresh: virtualNetworkStateRefreshFunc(client, resGroup, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Virtual Network (%s) to become available: %s", d.Id(), err)
	}

	return resourceArmVirtualNetworkRead(d, meta)
}

// resourceArmVirtualNetworkRead goes ahead and reads the state of the corresponding ARM virtual network.
func resourceArmVirtualNetworkRead(d *schema.ResourceData, meta interface{}) error {
	vnetClient := meta.(*ArmClient).vnetClient

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Sending virtual network read request to ARM.")

	resp, err := vnetClient.Get(resGroup, name)
	if resp.StatusCode == http.StatusNotFound {
		// it means the virtual network has been deleted in the meantime;
		// so we must go ahead and remove it here:
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure virtual network %s: %s", name, err)
	}
	vnet := *resp.Properties

	// update all the appropriate values:
	d.Set("address_space", vnet.AddressSpace.AddressPrefixes)

	// read state of subnets:
	subnets := &schema.Set{
		F: resourceAzureSubnetHash,
	}

	for _, subnet := range *vnet.Subnets {
		s := map[string]interface{}{}

		s["name"] = *subnet.Name
		s["address_prefix"] = *subnet.Properties.AddressPrefix
		// NOTE(aznashwan): ID's necessary?
		if subnet.Properties.NetworkSecurityGroup != nil {
			s["security_group"] = *subnet.Properties.NetworkSecurityGroup.ID
		}

		subnets.Add(s)
	}
	d.Set("subnet", subnets)

	// now; dns servers:
	dnses := []string{}
	for _, dns := range *vnet.DhcpOptions.DNSServers {
		dnses = append(dnses, dns)
	}
	d.Set("dns_servers_names", dnses)

	return nil
}

// resourceArmVirtualNetworkUpdate goes ahead and updates the corresponding ARM virtual network.
func resourceArmVirtualNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	// considering Create's idempotency, Update is simply a proxy for it...
	// Update has been left as a separate function here for utmost clarity:
	return resourceArmVirtualNetworkCreate(d, meta)
}

// resourceArmVirtualNetworkDelete deletes the specified ARM virtual network.
func resourceArmVirtualNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	vnetClient := meta.(*ArmClient).vnetClient

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)
	_, err := vnetClient.Delete(resGroup, name)

	return err
}

// getVirtualNetworkProperties is a helper function which returns the
// VirtualNetworkPropertiesFormat of the network resource.
func getVirtualNetworkProperties(d *schema.ResourceData) *network.VirtualNetworkPropertiesFormat {
	// first; get address space prefixes:
	prefixes := []string{}
	for _, prefix := range d.Get("address_space").([]interface{}) {
		prefixes = append(prefixes, prefix.(string))
	}

	// then; the dns servers:
	dnses := []string{}
	for _, dns := range d.Get("dns_servers_names").([]interface{}) {
		dnses = append(dnses, dns.(string))
	}

	// then; the subnets:
	subnets := []network.Subnet{}
	if subs := d.Get("subnet").(*schema.Set); subs.Len() > 0 {
		for _, subnet := range subs.List() {
			subnet := subnet.(map[string]interface{})

			name := subnet["name"].(string)
			prefix := subnet["address_prefix"].(string)
			secGroup := subnet["security_group"].(string)

			var subnetObj network.Subnet
			subnetObj.Name = &name
			subnetObj.Properties = &network.SubnetPropertiesFormat{}
			subnetObj.Properties.AddressPrefix = &prefix

			if secGroup != "" {
				subnetObj.Properties.NetworkSecurityGroup = &network.SubResource{
					ID: &secGroup,
				}
			}

			subnets = append(subnets, subnetObj)
		}
	}

	// finally; return the struct:
	return &network.VirtualNetworkPropertiesFormat{
		AddressSpace: &network.AddressSpace{
			AddressPrefixes: &prefixes,
		},
		DhcpOptions: &network.DhcpOptions{
			DNSServers: &dnses,
		},
		Subnets: &subnets,
	}
}

func resourceAzureSubnetHash(v interface{}) int {
	m := v.(map[string]interface{})
	subnet := m["name"].(string) + m["address_prefix"].(string)
	if securityGroup, present := m["security_group"]; present {
		subnet = subnet + securityGroup.(string)
	}
	return hashcode.String(subnet)
}

// virtualNetworkStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a virtual network.
func virtualNetworkStateRefreshFunc(client *ArmClient, resourceGroupName string, networkName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.vnetClient.Get(resourceGroupName, networkName)
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in virtualNetworkStateRefreshFunc to Azure ARM for virtual network '%s' (RG: '%s'): %s", networkName, resourceGroupName, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}
