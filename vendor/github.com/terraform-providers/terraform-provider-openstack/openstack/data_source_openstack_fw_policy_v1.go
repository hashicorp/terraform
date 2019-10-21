package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/fwaas/policies"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceFWPolicyV1() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceFWPolicyV1Read,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"policy_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"tenant_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"audited": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"shared": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"rules": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceFWPolicyV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	listOpts := policies.ListOpts{
		ID:       d.Get("policy_id").(string),
		Name:     d.Get("name").(string),
		TenantID: d.Get("tenant_id").(string),
	}

	pages, err := policies.List(networkingClient, listOpts).AllPages()
	if err != nil {
		return err
	}

	allFWPolicies, err := policies.ExtractPolicies(pages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve firewall policies: %s", err)
	}

	if len(allFWPolicies) < 1 {
		return fmt.Errorf("No firewall policies found with name: %s", d.Get("name"))
	}

	if len(allFWPolicies) > 1 {
		return fmt.Errorf("More than one firewall policies found with name: %s", d.Get("name"))
	}

	policy := allFWPolicies[0]

	log.Printf("[DEBUG] Retrieved firewall policies %s: %+v", policy.ID, policy)
	d.SetId(policy.ID)

	d.Set("name", policy.Name)
	d.Set("tenant_id", policy.TenantID)
	d.Set("description", policy.Description)
	d.Set("audited", policy.Audited)
	d.Set("shared", policy.Shared)
	d.Set("rules", policy.Rules)
	d.Set("region", GetRegion(d, config))

	return nil
}
