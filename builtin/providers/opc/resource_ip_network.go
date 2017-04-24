package opc

import (
	"fmt"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOPCIPNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCIPNetworkCreate,
		Read:   resourceOPCIPNetworkRead,
		Update: resourceOPCIPNetworkUpdate,
		Delete: resourceOPCIPNetworkDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"ip_address_prefix": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateIPPrefixCIDR,
			},

			"ip_network_exchange": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"public_napt_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsOptionalSchema(),
		},
	}
}

func resourceOPCIPNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPNetworks()

	// Get required attributes
	name := d.Get("name").(string)
	ipPrefix := d.Get("ip_address_prefix").(string)
	// public_napt_enabled is not required, but bool type allows it to be unspecified
	naptEnabled := d.Get("public_napt_enabled").(bool)

	input := &compute.CreateIPNetworkInput{
		Name:              name,
		IPAddressPrefix:   ipPrefix,
		PublicNaptEnabled: naptEnabled,
	}

	// Get Optional attributes
	if desc, ok := d.GetOk("description"); ok && desc != nil {
		input.Description = desc.(string)
	}

	if ipEx, ok := d.GetOk("ip_network_exchange"); ok && ipEx != nil {
		input.IPNetworkExchange = ipEx.(string)
	}

	tags := getStringList(d, "tags")
	if len(tags) != 0 {
		input.Tags = tags
	}

	info, err := client.CreateIPNetwork(input)
	if err != nil {
		return fmt.Errorf("Error creating IP Network '%s': %v", name, err)
	}

	d.SetId(info.Name)

	return resourceOPCIPNetworkRead(d, meta)
}

func resourceOPCIPNetworkRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPNetworks()

	name := d.Id()
	input := &compute.GetIPNetworkInput{
		Name: name,
	}

	res, err := client.GetIPNetwork(input)
	if err != nil {
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IP Network '%s': %v", name, err)
	}

	d.Set("name", res.Name)
	d.Set("ip_address_prefix", res.IPAddressPrefix)
	d.Set("ip_network_exchanged", res.IPNetworkExchange)
	d.Set("description", res.Description)
	d.Set("public_napt_enabled", res.PublicNaptEnabled)
	d.Set("uri", res.Uri)
	if err := setStringList(d, "tags", res.Tags); err != nil {
		return err
	}
	return nil
}

func resourceOPCIPNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPNetworks()

	// Get required attributes
	name := d.Get("name").(string)
	ipPrefix := d.Get("ip_address_prefix").(string)
	// public_napt_enabled is not required, but bool type allows it to be unspecified
	naptEnabled := d.Get("public_napt_enabled").(bool)

	input := &compute.UpdateIPNetworkInput{
		Name:              name,
		IPAddressPrefix:   ipPrefix,
		PublicNaptEnabled: naptEnabled,
	}

	// Get Optional attributes
	desc, descOk := d.GetOk("description")
	if descOk {
		input.Description = desc.(string)
	}

	ipEx, ipExOk := d.GetOk("ip_network_exchange")
	if ipExOk {
		input.IPNetworkExchange = ipEx.(string)
	}

	tags := getStringList(d, "tags")
	if len(tags) != 0 {
		input.Tags = tags
	}

	info, err := client.UpdateIPNetwork(input)
	if err != nil {
		return fmt.Errorf("Error updating IP Network '%s': %v", name, err)
	}

	d.SetId(info.Name)

	return resourceOPCIPNetworkRead(d, meta)
}

func resourceOPCIPNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPNetworks()

	name := d.Id()
	input := &compute.DeleteIPNetworkInput{
		Name: name,
	}

	if err := client.DeleteIPNetwork(input); err != nil {
		return fmt.Errorf("Error deleting IP Network '%s': %v", name, err)
	}
	return nil
}
