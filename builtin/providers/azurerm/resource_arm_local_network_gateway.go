package azurerm

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/core/http"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmLocalNetworkGateway returns the schema.Resource
// associated to an Azure local network gateway.
func resourceArmLocalNetworkGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLocalNetworkGatewayCreate,
		Read:   resourceArmLocalNetworkGatewayRead,
		Update: resourceArmLocalNetworkGatewayUpdate,
		Delete: resourceArmLocalNetworkGatewayDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"resource_guid": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"gateway_address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"address_space": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

// resourceArmLocalNetworkGatewayCreate goes ahead and creates the specified ARM local network gateway.
func resourceArmLocalNetworkGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	lnetClient := meta.(*ArmClient).localNetConnClient

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	ipAddress := d.Get("gateway_address").(string)

	// NOTE: due to the including-but-different relationship between the ASM
	// and ARM APIs, one may set the following local network gateway type to
	// "Classic" and basically get an old ASM local network connection through
	// the ARM API. This functionality is redundant with respect to the old
	// ASM-based implementation which we already have, so we just use the
	// new Resource Manager APIs here:
	typ := "Resource Manager"

	// fetch the 'address_space_prefix'es:
	prefixes := []string{}
	for _, pref := range d.Get("addres_space").([]interface{}) {
		prefixes = append(prefixes, pref.(string))
	}

	// NOTE: result ignored here; review below...
	resp, err := lnetClient.CreateOrUpdate(resGroup, name, network.LocalNetworkGateway{
		Name:     &name,
		Location: &location,
		Type:     &typ,
		Properties: &network.LocalNetworkGatewayPropertiesFormat{
			LocalNetworkAddressSpace: &network.AddressSpace{
				AddressPrefixes: &prefixes,
			},
			GatewayIPAddress: &ipAddress,
		},
	})
	if err != nil {
		return fmt.Errorf("Error reading the state of Azure ARM Local Network Gateway '%s': %s", name, err)
	}

	// NOTE: we either call read here or basically repeat the reading process
	// with the ignored network.LocalNetworkGateway result of the above:
	d.SetId(*resp.ID)
	return resourceArmLocalNetworkGatewayRead(d, meta)
}

// resourceArmLocalNetworkGatewayRead goes ahead and reads the state of the corresponding ARM local network gateway.
func resourceArmLocalNetworkGatewayRead(d *schema.ResourceData, meta interface{}) error {
	lnetClient := meta.(*ArmClient).localNetConnClient

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Sending GET request to Azure ARM for local network gateway '%s'.", name)
	lnet, err := lnetClient.Get(resGroup, name)
	if lnet.StatusCode == http.StatusNotFound {
		// it means that the resource has been deleted in the meantime...
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error reading the state of Azure ARM local network gateway '%s': %s", name, err)
	}

	d.Set("resource_guid", *lnet.Properties.ResourceGUID)
	d.Set("gateway_address", *lnet.Properties.GatewayIPAddress)

	prefs := []string{}
	if ps := *lnet.Properties.LocalNetworkAddressSpace.AddressPrefixes; ps != nil {
		prefs = ps
	}
	d.Set("address_space", prefs)

	return nil
}

// resourceArmLocalNetworkGatewayUpdate goes ahead and updates the corresponding ARM local network gateway.
func resourceArmLocalNetworkGatewayUpdate(d *schema.ResourceData, meta interface{}) error {
	// NOTE: considering the idempotency, we can safely call create again on
	// update. This has been written out in order to ensure clarity,
	return resourceArmLocalNetworkGatewayCreate(d, meta)
}

// resourceArmLocalNetworkGatewayDelete deletes the specified ARM local network gateway.
func resourceArmLocalNetworkGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	lnetClient := meta.(*ArmClient).localNetConnClient

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Sending Azure ARM delete request for local network gateway '%s'.", name)
	_, err := lnetClient.Delete(resGroup, name)
	if err != nil {
		return fmt.Errorf("Error issuing Azure ARM delete request of local network gateway '%s': %s", name, err)
	}

	return nil
}
