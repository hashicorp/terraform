package azure

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmLocalNetworkConnectionCreate goes ahead and creates the specified ARM local network connection.
func resourceArmLocalNetworkConnectionCreate(d *schema.ResourceData, meta interface{}) error {
	lnetClient := meta.(*AzureClient).armClient.localNetConnClient

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	ipAddress := d.Get("vpn_gateway_address").(string)

	// NOTE: due to the including-but-different relationship between the ASM
	// and ARM APIs, one may set the following local network connection type to
	// "Classic" and basically get an old ASM local network connection through
	// the ARM API. This functionality is redundant with respect to the old
	// ASM-based implementation which we already have, so we just spare the
	// users of having to put in another resource attribute for this and simply
	// use the new ARM APIs here:
	typ := "Resource Manager"

	// fetch the 'address_space_prefix'es:
	prefixes := []string{}
	for _, pref := range d.Get("addres_space_prefixes").([]interface{}) {
		prefixes = append(prefixes, pref.(string))
	}

	// NOTE: result ignored here; review below...
	if _, err := lnetClient.CreateOrUpdate(resGroup, name, network.LocalNetworkGateway{
		Name:     &name,
		Location: &location,
		Type:     &typ,
		Properties: &network.LocalNetworkGatewayPropertiesFormat{
			LocalNetworkAddressSpace: &network.AddressSpace{
				AddressPrefixes: &prefixes,
			},
			GatewayIPAddress: &ipAddress,
		},
	}); err != nil {
		return fmt.Errorf("Error reading the state of Azure ARM Local Network Gateway '%s': %s", name, err)
	}

	// NOTE: we either call read here or basically repeat the reading process
	// with the ignored network.LocalNetworkGateway result of the above.
	d.SetId(name)
	return resourceArmLocalNetworkConnectionRead(d, meta)
}

// resourceArmLocalNetworkConnectionRead goes ahead and reads the state of the corresponding ARM local network connection.
func resourceArmLocalNetworkConnectionRead(d *schema.ResourceData, meta interface{}) error {
	lnetClient := meta.(*AzureClient).armClient.localNetConnClient

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Sending GET request to Azure ARM for local network gateway '%s'.", name)
	lnet, err := lnetClient.Get(resGroup, name)
	if err != nil {
		// NOTE: the direct return of the error here is absolutely vital for
		// the resourceArmLocalNetworkExists function:
		return err
	}

	d.Set("vpn_gateway_address", *lnet.Properties.GatewayIPAddress)

	prefs := []string{}
	if ps := *lnet.Properties.LocalNetworkAddressSpace.AddressPrefixes; ps != nil {
		prefs = ps
	}
	d.Set("address_space_prefixes", prefs)

	return nil
}

// resourceArmLocalNetworkConnectionUpdate goes ahead and updates the corresponding ARM local network connection.
func resourceArmLocalNetworkConnectionUpdate(d *schema.ResourceData, meta interface{}) error {
	// NOTE: considering the idempotency, we can safely call create again on
	// update. This has been written out in order to ensure clarity,
	return resourceArmLocalNetworkConnectionCreate(d, meta)
}

// resourceArmLocalNetworkExists goes ahead and checks whether or not the given ARM local network connection exists.
func resourceArmLocalNetworkConnectionExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	if err := resourceArmLocalNetworkConnectionRead(d, meta); err != nil {
		return true, nil
	}

	return false, nil
}

// resourceArmLocalNetworkConnectionDelete deletes the specified ARM local network connection.
func resourceArmLocalNetworkConnectionDelete(d *schema.ResourceData, meta interface{}) error {
	lnetClient := meta.(*AzureClient).armClient.localNetConnClient

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Sending Azure ARM delete request for local network connection '%s'.", name)
	_, err := lnetClient.Delete(resGroup, name)
	if err != nil {
		return fmt.Errorf("Error issuing Azure ARM delete request of local network gateway '%s': %s", name, err)
	}

	return nil
}
