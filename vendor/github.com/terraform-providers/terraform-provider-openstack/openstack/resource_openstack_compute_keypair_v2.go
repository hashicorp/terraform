package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceComputeKeypairV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeKeypairV2Create,
		Read:   resourceComputeKeypairV2Read,
		Delete: resourceComputeKeypairV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
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
			"value_specs": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceComputeKeypairV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	createOpts := KeyPairCreateOpts{
		keypairs.CreateOpts{
			Name:      d.Get("name").(string),
			PublicKey: d.Get("public_key").(string),
		},
		MapValueSpecs(d),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	kp, err := keypairs.Create(computeClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack keypair: %s", err)
	}

	d.SetId(kp.Name)

	return resourceComputeKeypairV2Read(d, meta)
}

func resourceComputeKeypairV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	kp, err := keypairs.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "keypair")
	}

	d.Set("name", kp.Name)
	d.Set("public_key", kp.PublicKey)
	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceComputeKeypairV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	err = keypairs.Delete(computeClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack keypair: %s", err)
	}
	d.SetId("")
	return nil
}
