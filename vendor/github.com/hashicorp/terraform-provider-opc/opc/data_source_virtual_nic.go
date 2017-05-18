package opc

import (
	"fmt"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceVNIC() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVNICRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"mac_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsComputedSchema(),

			"transit_flag": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceVNICRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).VirtNICs()

	name := d.Get("name").(string)

	input := &compute.GetVirtualNICInput{
		Name: name,
	}

	vnic, err := client.GetVirtualNIC(input)
	if err != nil {
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading vnic %s: %s", name, err)
	}

	d.SetId(name)
	d.Set("description", vnic.Description)
	d.Set("mac_address", vnic.MACAddress)
	d.Set("transit_flag", vnic.TransitFlag)
	d.Set("uri", vnic.Uri)
	if err := setStringList(d, "tags", vnic.Tags); err != nil {
		return err
	}
	return nil
}
