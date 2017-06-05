package azurerm

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceArmPublicIP() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceArmPublicIPRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceArmPublicIPRead(d *schema.ResourceData, meta interface{}) error {
	publicIPClient := meta.(*ArmClient).publicIPClient

	resGroup := d.Get("resource_group_name").(string)
	name := d.Get("name").(string)

	resp, err := publicIPClient.Get(resGroup, name, "")
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure public ip %s: %s", name, err)
	}

	d.SetId(*resp.ID)
	d.Set("location", resp.Location)
	d.Set("public_ip_address_allocation", strings.ToLower(string(resp.PublicIPAddressPropertiesFormat.PublicIPAllocationMethod)))

	if resp.PublicIPAddressPropertiesFormat.DNSSettings != nil && resp.PublicIPAddressPropertiesFormat.DNSSettings.Fqdn != nil && *resp.PublicIPAddressPropertiesFormat.DNSSettings.Fqdn != "" {
		d.Set("fqdn", resp.PublicIPAddressPropertiesFormat.DNSSettings.Fqdn)
	}

	if resp.PublicIPAddressPropertiesFormat.IPAddress != nil && *resp.PublicIPAddressPropertiesFormat.IPAddress != "" {
		d.Set("ip_address", resp.PublicIPAddressPropertiesFormat.IPAddress)
	}

	flattenAndSetTags(d, resp.Tags)
	return nil
}
