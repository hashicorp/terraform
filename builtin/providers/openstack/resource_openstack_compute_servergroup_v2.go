package openstack

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/servergroups"
)

func resourceComputeServerGroupV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeServerGroupV2Create,
		Read:   resourceComputeServerGroupV2Read,
		Update: nil,
		Delete: resourceComputeServerGroupV2Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"policies": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"members": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceComputeServerGroupV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	createOpts := &servergroups.CreateOpts{
		Name:     d.Get("name").(string),
		Policies: resourceServerGroupPoliciesV2(d),
	}
	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	newSG, err := servergroups.Create(computeClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating ServerGroup: %s", err)
	}

	d.SetId(newSG.ID)

	return resourceComputeServerGroupV2Read(d, meta)
}

func resourceComputeServerGroupV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	sg, err := servergroups.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "server group")
	}

	log.Printf("[DEBUG] Retrieved ServerGroup %s: %+v", d.Id(), sg)

	// Set the name
	d.Set("name", sg.Name)

	// Set the policies
	policies := []string{}
	for _, p := range sg.Policies {
		policies = append(policies, p)
	}
	d.Set("policies", policies)

	// Set the members
	members := []string{}
	for _, m := range sg.Members {
		members = append(members, m)
	}
	d.Set("members", members)

	return nil
}

func resourceComputeServerGroupV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	log.Printf("[DEBUG] Deleting ServerGroup %s", d.Id())
	if err := servergroups.Delete(computeClient, d.Id()).ExtractErr(); err != nil {
		return fmt.Errorf("Error deleting ServerGroup: %s", err)
	}

	return nil
}

func resourceServerGroupPoliciesV2(d *schema.ResourceData) []string {
	rawPolicies := d.Get("policies").([]interface{})
	policies := make([]string, len(rawPolicies))
	for i, raw := range rawPolicies {
		policies[i] = raw.(string)
	}
	return policies
}
