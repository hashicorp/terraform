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
		Update: resourceArmVirtualNetworkCreate,
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

			"dns_servers": &schema.Schema{
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

	resp, err := vnetClient.CreateOrUpdate(resGroup, name, vnet)
	if err != nil {
		return err
	}

	d.SetId(*resp.ID)

	log.Printf("[DEBUG] Waiting for Virtual Network (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  "Succeeded",
		Refresh: virtualNetworkStateRefreshFunc(client, resGroup, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Virtual Network (%s) to become available: %s", name, err)
	}

	return resourceArmVirtualNetworkRead(d, meta)
}

func resourceArmVirtualNetworkRead(d *schema.ResourceData, meta interface{}) error {
	vnetClient := meta.(*ArmClient).vnetClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["virtualNetworks"]

	resp, err := vnetClient.Get(resGroup, name)
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure virtual network %s: %s", name, err)
	}
	vnet := *resp.Properties

	// update appropriate values
	d.Set("address_space", vnet.AddressSpace.AddressPrefixes)

	subnets := &schema.Set{
		F: resourceAzureSubnetHash,
	}

	for _, subnet := range *vnet.Subnets {
		s := map[string]interface{}{}

		s["name"] = *subnet.Name
		s["address_prefix"] = *subnet.Properties.AddressPrefix
		if subnet.Properties.NetworkSecurityGroup != nil {
			s["security_group"] = *subnet.Properties.NetworkSecurityGroup.ID
		}

		subnets.Add(s)
	}
	d.Set("subnet", subnets)

	dnses := []string{}
	for _, dns := range *vnet.DhcpOptions.DNSServers {
		dnses = append(dnses, dns)
	}
	d.Set("dns_servers", dnses)

	return nil
}

func resourceArmVirtualNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	vnetClient := meta.(*ArmClient).vnetClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["virtualNetworks"]

	_, err = vnetClient.Delete(resGroup, name)

	return err
}

func getVirtualNetworkProperties(d *schema.ResourceData) *network.VirtualNetworkPropertiesFormat {
	// first; get address space prefixes:
	prefixes := []string{}
	for _, prefix := range d.Get("address_space").([]interface{}) {
		prefixes = append(prefixes, prefix.(string))
	}

	// then; the dns servers:
	dnses := []string{}
	for _, dns := range d.Get("dns_servers").([]interface{}) {
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

func virtualNetworkStateRefreshFunc(client *ArmClient, resourceGroupName string, networkName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.vnetClient.Get(resourceGroupName, networkName)
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in virtualNetworkStateRefreshFunc to Azure ARM for virtual network '%s' (RG: '%s'): %s", networkName, resourceGroupName, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}
