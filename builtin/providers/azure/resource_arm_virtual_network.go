package azure

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmVirtualNetworkCreate goes ahead and creates the specified ARM virtual network.
func resourceArmVirtualNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	vnetClient := meta.(*AzureClient).armClient.vnetClient

	log.Printf("[INFO] preparing arguments for Azure ARM virtual network creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)

	// NOTE: due to the including-but-different relationship between the ASM
	// and ARM APIs, one may set the following virtual network type to
	// "Classic" and basically get an old ASM virtual network through the ARM
	// API. This functionality is redundant with respect to the old ASM-based
	// implementation which we already have, so we just spare the users of
	// having to put in another resource attribute for this and simply use the
	// new ARM APIs here:
	vnetType := "Resource Manager"

	vnet := network.VirtualNetwork{
		Name:       &name,
		Location:   &location,
		Type:       &vnetType,
		Properties: getVirtualNetworkProperties(d),
	}

	log.Printf("[INFO] ")
	res, err := vnetClient.CreateOrUpdate(resGroup, name, vnet)
	if err != nil {
		return err
	}

	if res.Response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Creation request was denies: code: %d", res.Response.StatusCode)
	}

	return nil
}

// resourceArmVirtualNetworkRead goes ahead and reads the state of the corresponding ARM virtual network.
func resourceArmVirtualNetworkRead(d *schema.ResourceData, meta interface{}) error {
	vnetClient := meta.(*AzureClient).armClient.vnetClient

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
		s["security_group"] = *subnet.Properties.NetworkSecurityGroup.ID

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
	vnetClient := meta.(*AzureClient).armClient.vnetClient

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

			subnets = append(subnets, network.Subnet{
				Name: &name,
				Properties: &network.SubnetPropertiesFormat{
					AddressPrefix: &prefix,
					NetworkSecurityGroup: &network.SubResource{
						ID: &secGroup,
					},
				},
			})
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
