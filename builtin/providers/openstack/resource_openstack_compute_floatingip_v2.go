package openstack

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/floatingip"
)

func resourceComputeFloatingIPV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeFloatingIPV2Create,
		Read:   resourceComputeFloatingIPV2Read,
		Update: nil,
		Delete: resourceComputeFloatingIPV2Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFuncAllowMissing("OS_REGION_NAME"),
			},

			"pool": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFunc("OS_POOL_NAME"),
			},

			"address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"fixed_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"instance_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeFloatingIPV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	createOpts := &floatingip.CreateOpts{
		Pool: d.Get("pool").(string),
	}
	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	newFip, err := floatingip.Create(computeClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating Floating IP: %s", err)
	}

	d.SetId(newFip.ID)

	return resourceComputeFloatingIPV2Read(d, meta)
}

func resourceComputeFloatingIPV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	fip, err := floatingip.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "floating ip")
	}

	log.Printf("[DEBUG] Retrieved Floating IP %s: %+v", d.Id(), fip)

	d.Set("pool", fip.Pool)
	d.Set("instance_id", fip.InstanceID)
	d.Set("address", fip.IP)
	d.Set("fixed_ip", fip.FixedIP)

	return nil
}

func resourceComputeFloatingIPV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	log.Printf("[DEBUG] Deleting Floating IP %s", d.Id())
	if err := floatingip.Delete(computeClient, d.Id()).ExtractErr(); err != nil {
		return fmt.Errorf("Error deleting Floating IP: %s", err)
	}

	return nil
}
