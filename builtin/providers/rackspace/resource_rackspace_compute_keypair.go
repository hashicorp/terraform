package rackspace

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	osKeypairs "github.com/rackspace/gophercloud/openstack/compute/v2/extensions/keypairs"
	rsKeypairs "github.com/rackspace/gophercloud/rackspace/compute/v2/keypairs"
)

func resourceComputeKeypair() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeKeypairCreate,
		Read:   resourceComputeKeypairRead,
		Delete: resourceComputeKeypairDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFunc("RS_REGION_NAME"),
			},
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
	computeClient, err := config.computeClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace compute client: %s", err)
	}

	createOpts := osKeypairs.CreateOpts{
		Name:      d.Get("name").(string),
		PublicKey: d.Get("public_key").(string),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	kp, err := rsKeypairs.Create(computeClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating Rackspace keypair: %s", err)
	}

	d.SetId(kp.Name)

	return resourceComputeKeypairRead(d, meta)
}

func resourceComputeKeypairRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace compute client: %s", err)
	}

	kp, err := rsKeypairs.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "keypair")
	}

	d.Set("name", kp.Name)
	d.Set("public_key", kp.PublicKey)

	return nil
}

func resourceComputeKeypairDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace compute client: %s", err)
	}

	err = rsKeypairs.Delete(computeClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting Rackspace keypair: %s", err)
	}
	d.SetId("")
	return nil
}
