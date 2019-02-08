package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceComputeServerGroupV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeServerGroupV2Create,
		Read:   resourceComputeServerGroupV2Read,
		Update: nil,
		Delete: resourceComputeServerGroupV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"policies": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"members": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"value_specs": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceComputeServerGroupV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	name := d.Get("name").(string)

	rawPolicies := d.Get("policies").([]interface{})
	policies := expandComputeServerGroupV2Policies(computeClient, rawPolicies)

	createOpts := ComputeServerGroupV2CreateOpts{
		servergroups.CreateOpts{
			Name:     name,
			Policies: policies,
		},
		MapValueSpecs(d),
	}

	log.Printf("[DEBUG] openstack_compute_servergroup_v2 create options: %#v", createOpts)
	newSG, err := servergroups.Create(computeClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating openstack_compute_servergroup_v2 %s: %s", name, err)
	}

	d.SetId(newSG.ID)

	return resourceComputeServerGroupV2Read(d, meta)
}

func resourceComputeServerGroupV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	sg, err := servergroups.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "Error retrieving openstack_compute_servergroup_v2")
	}

	log.Printf("[DEBUG] Retrieved openstack_compute_servergroup_v2 %s: %#v", d.Id(), sg)

	d.Set("name", sg.Name)
	d.Set("policies", sg.Policies)
	d.Set("members", sg.Members)

	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceComputeServerGroupV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	if err := servergroups.Delete(computeClient, d.Id()).ExtractErr(); err != nil {
		return CheckDeleted(d, err, "Error deleting openstack_compute_servergroup_v2")
	}

	return nil
}
