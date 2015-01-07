package openstack

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/keypairs"
)

func resourceComputeKeypair() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeKeypairCreate,
		Read:   resourceComputeKeypairRead,
		Delete: resourceComputeKeypairDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"public_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceComputeKeypairCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.computeV2Client

	createOpts := keypairs.CreateOpts{
		Name:      d.Get("name").(string),
		PublicKey: d.Get("public_key").(string),
	}

	kp, err := keypairs.Create(osClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack keypair: %s", err)
	}

	d.SetId(kp.Name)

	return resourceComputeKeypairRead(d, meta)
}

func resourceComputeKeypairRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.computeV2Client

	kp, err := keypairs.Get(osClient, d.Id()).Extract()
	if err != nil {
		return fmt.Errorf("Error retrieving OpenStack keypair: %s", err)
	}

	d.Set("name", kp.Name)
	d.Set("public_key", kp.PublicKey)

	return nil
}

func resourceComputeKeypairDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.computeV2Client

	err := keypairs.Delete(osClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack keypair: %s", err)
	}
	d.SetId("")
	return nil
}
