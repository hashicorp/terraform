package openstack

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceComputeSecGroupV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeSecGroupV2Create,
		Read:   resourceComputeSecGroupV2Read,
		Update: resourceComputeSecGroupV2Update,
		Delete: resourceComputeSecGroupV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(10 * time.Minute),
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
				Required: true,
				ForceNew: false,
			},

			"description": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"rule": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Set:      computeSecGroupV2RuleHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"from_port": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: false,
						},

						"to_port": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: false,
						},

						"ip_protocol": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},

						"cidr": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
							StateFunc: func(v interface{}) string {
								return strings.ToLower(v.(string))
							},
						},

						"from_group_id": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},

						"self": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
							ForceNew: false,
						},
					},
				},
			},
		},
	}
}

func resourceComputeSecGroupV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	// Before creating the security group, make sure all rules are valid.
	if err := computeSecGroupV2RulesCheckForErrors(d); err != nil {
		return err
	}

	// If all rules are valid, proceed with creating the security gruop.
	name := d.Get("name").(string)
	createOpts := secgroups.CreateOpts{
		Name:        name,
		Description: d.Get("description").(string),
	}

	log.Printf("[DEBUG] openstack_compute_secgroup_v2 Create Options: %#v", createOpts)
	sg, err := secgroups.Create(computeClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating openstack_compute_secgroup_v2 %s: %s", name, err)
	}

	d.SetId(sg.ID)

	// Now that the security group has been created, iterate through each rule and create it
	createRuleOptsList := expandComputeSecGroupV2CreateRules(d)

	for _, createRuleOpts := range createRuleOptsList {
		_, err := secgroups.CreateRule(computeClient, createRuleOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error creating openstack_compute_secgroup_v2 %s rule: %s", name, err)
		}
	}

	return resourceComputeSecGroupV2Read(d, meta)
}

func resourceComputeSecGroupV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	sg, err := secgroups.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "Error retrieving openstack_compute_secgroup_v2")
	}

	d.Set("name", sg.Name)
	d.Set("description", sg.Description)

	rules, err := flattenComputeSecGroupV2Rules(computeClient, d, sg.Rules)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Retrieved openstack_compute_secgroup_v2 %s rules: %#v", d.Id(), rules)

	if err := d.Set("rule", rules); err != nil {
		return fmt.Errorf("Unable to set openstack_compute_secgroup_v2 %s rules: %s", d.Id(), err)
	}

	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceComputeSecGroupV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	description := d.Get("description").(string)
	updateOpts := secgroups.UpdateOpts{
		Name:        d.Get("name").(string),
		Description: &description,
	}

	log.Printf("[DEBUG] openstack_compute_secgroup_v2 %s Update Options: %#v", d.Id(), updateOpts)

	_, err = secgroups.Update(computeClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating openstack_compute_secgroup_v2 %s: %s", d.Id(), err)
	}

	if d.HasChange("rule") {
		oldSGRaw, newSGRaw := d.GetChange("rule")
		oldSGRSet, newSGRSet := oldSGRaw.(*schema.Set), newSGRaw.(*schema.Set)
		secgrouprulesToAdd := newSGRSet.Difference(oldSGRSet)
		secgrouprulesToRemove := oldSGRSet.Difference(newSGRSet)

		log.Printf("[DEBUG] openstack_compute_secgroup_v2 %s rules to add: %v", d.Id(), secgrouprulesToAdd)
		log.Printf("[DEBUG] openstack_compute_secgroup_v2 %s rules to remove: %v", d.Id(), secgrouprulesToRemove)

		for _, rawRule := range secgrouprulesToAdd.List() {
			createRuleOpts := expandComputeSecGroupV2CreateRule(d, rawRule)

			_, err := secgroups.CreateRule(computeClient, createRuleOpts).Extract()
			if err != nil {
				return fmt.Errorf("Error adding rule to openstack_compute_secgroup_v2 %s: %s", d.Id(), err)
			}
		}

		for _, r := range secgrouprulesToRemove.List() {
			rule := expandComputeSecGroupV2Rule(d, r)

			err := secgroups.DeleteRule(computeClient, rule.ID).ExtractErr()
			if err != nil {
				if _, ok := err.(gophercloud.ErrDefault404); ok {
					continue
				}

				return fmt.Errorf("Error removing rule %s from openstack_compute_secgroup_v2 %s: %s", rule.ID, d.Id(), err)
			}
		}
	}

	return resourceComputeSecGroupV2Read(d, meta)
}

func resourceComputeSecGroupV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    computeSecGroupV2StateRefreshFunc(computeClient, d),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return CheckDeleted(d, err, "Error deleting openstack_compute_secgroup_v2")
	}

	return nil
}
