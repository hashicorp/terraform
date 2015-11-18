package openstack

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
)

func resourceComputeSecGroupV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeSecGroupV2Create,
		Read:   resourceComputeSecGroupV2Read,
		Update: resourceComputeSecGroupV2Update,
		Delete: resourceComputeSecGroupV2Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFuncAllowMissing("OS_REGION_NAME"),
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"rule": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"from_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: false,
						},
						"to_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: false,
						},
						"ip_protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},
						"cidr": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},
						"from_group_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},
						"self": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
							ForceNew: false,
						},
					},
				},
				Set: secgroupRuleV2Hash,
			},
		},
	}
}

func resourceComputeSecGroupV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	createOpts := secgroups.CreateOpts{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	sg, err := secgroups.Create(computeClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack security group: %s", err)
	}

	d.SetId(sg.ID)

	createRuleOptsList := resourceSecGroupRulesV2(d)
	for _, createRuleOpts := range createRuleOptsList {
		_, err := secgroups.CreateRule(computeClient, createRuleOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error creating OpenStack security group rule: %s", err)
		}
	}

	return resourceComputeSecGroupV2Read(d, meta)
}

func resourceComputeSecGroupV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	sg, err := secgroups.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "security group")
	}

	d.Set("name", sg.Name)
	d.Set("description", sg.Description)

	rtm, err := rulesToMap(computeClient, d, sg.Rules)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] rulesToMap(sg.Rules): %+v", rtm)
	d.Set("rule", rtm)

	return nil
}

func resourceComputeSecGroupV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	updateOpts := secgroups.UpdateOpts{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	log.Printf("[DEBUG] Updating Security Group (%s) with options: %+v", d.Id(), updateOpts)

	_, err = secgroups.Update(computeClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack security group (%s): %s", d.Id(), err)
	}

	if d.HasChange("rule") {
		oldSGRaw, newSGRaw := d.GetChange("rule")
		oldSGRSet, newSGRSet := oldSGRaw.(*schema.Set), newSGRaw.(*schema.Set)
		secgrouprulesToAdd := newSGRSet.Difference(oldSGRSet)
		secgrouprulesToRemove := oldSGRSet.Difference(newSGRSet)

		log.Printf("[DEBUG] Security group rules to add: %v", secgrouprulesToAdd)
		log.Printf("[DEBUG] Security groups rules to remove: %v", secgrouprulesToRemove)

		for _, rawRule := range secgrouprulesToAdd.List() {
			createRuleOpts := resourceSecGroupRuleCreateOptsV2(d, rawRule)
			rule, err := secgroups.CreateRule(computeClient, createRuleOpts).Extract()
			if err != nil {
				return fmt.Errorf("Error adding rule to OpenStack security group (%s): %s", d.Id(), err)
			}
			log.Printf("[DEBUG] Added rule (%s) to OpenStack security group (%s) ", rule.ID, d.Id())
		}

		for _, r := range secgrouprulesToRemove.List() {
			rule := resourceSecGroupRuleV2(d, r)
			err := secgroups.DeleteRule(computeClient, rule.ID).ExtractErr()
			if err != nil {
				errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
				if !ok {
					return fmt.Errorf("Error removing rule (%s) from OpenStack security group (%s): %s", rule.ID, d.Id(), err)
				}
				if errCode.Actual == 404 {
					continue
				} else {
					return fmt.Errorf("Error removing rule (%s) from OpenStack security group (%s)", rule.ID, d.Id())
				}
			} else {
				log.Printf("[DEBUG] Removed rule (%s) from OpenStack security group (%s): %s", rule.ID, d.Id(), err)
			}
		}
	}

	return resourceComputeSecGroupV2Read(d, meta)
}

func resourceComputeSecGroupV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     "DELETED",
		Refresh:    SecGroupV2StateRefreshFunc(computeClient, d),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack security group: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceSecGroupRulesV2(d *schema.ResourceData) []secgroups.CreateRuleOpts {
	rawRules := d.Get("rule").(*schema.Set).List()
	createRuleOptsList := make([]secgroups.CreateRuleOpts, len(rawRules))
	for i, rawRule := range rawRules {
		createRuleOptsList[i] = resourceSecGroupRuleCreateOptsV2(d, rawRule)
	}
	return createRuleOptsList
}

func resourceSecGroupRuleCreateOptsV2(d *schema.ResourceData, rawRule interface{}) secgroups.CreateRuleOpts {
	rawRuleMap := rawRule.(map[string]interface{})
	groupId := rawRuleMap["from_group_id"].(string)
	if rawRuleMap["self"].(bool) {
		groupId = d.Id()
	}
	return secgroups.CreateRuleOpts{
		ParentGroupID: d.Id(),
		FromPort:      rawRuleMap["from_port"].(int),
		ToPort:        rawRuleMap["to_port"].(int),
		IPProtocol:    rawRuleMap["ip_protocol"].(string),
		CIDR:          rawRuleMap["cidr"].(string),
		FromGroupID:   groupId,
	}
}

func resourceSecGroupRuleV2(d *schema.ResourceData, rawRule interface{}) secgroups.Rule {
	rawRuleMap := rawRule.(map[string]interface{})
	return secgroups.Rule{
		ID:            rawRuleMap["id"].(string),
		ParentGroupID: d.Id(),
		FromPort:      rawRuleMap["from_port"].(int),
		ToPort:        rawRuleMap["to_port"].(int),
		IPProtocol:    rawRuleMap["ip_protocol"].(string),
		IPRange:       secgroups.IPRange{CIDR: rawRuleMap["cidr"].(string)},
	}
}

func rulesToMap(computeClient *gophercloud.ServiceClient, d *schema.ResourceData, sgrs []secgroups.Rule) ([]map[string]interface{}, error) {
	sgrMap := make([]map[string]interface{}, len(sgrs))
	for i, sgr := range sgrs {
		groupId := ""
		self := false
		if sgr.Group.Name != "" {
			if sgr.Group.Name == d.Get("name").(string) {
				self = true
			} else {
				// Since Nova only returns the secgroup Name (and not the ID) for the group attribute,
				// we need to look up all security groups and match the name.
				// Nevermind that Nova wants the ID when setting the Group *and* that multiple groups
				// with the same name can exist...
				allPages, err := secgroups.List(computeClient).AllPages()
				if err != nil {
					return nil, err
				}
				securityGroups, err := secgroups.ExtractSecurityGroups(allPages)
				if err != nil {
					return nil, err
				}

				for _, sg := range securityGroups {
					if sg.Name == sgr.Group.Name {
						groupId = sg.ID
					}
				}
			}
		}

		sgrMap[i] = map[string]interface{}{
			"id":            sgr.ID,
			"from_port":     sgr.FromPort,
			"to_port":       sgr.ToPort,
			"ip_protocol":   sgr.IPProtocol,
			"cidr":          sgr.IPRange.CIDR,
			"self":          self,
			"from_group_id": groupId,
		}
	}
	return sgrMap, nil
}

func secgroupRuleV2Hash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["from_port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["to_port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["ip_protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cidr"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["from_group_id"].(string)))
	buf.WriteString(fmt.Sprintf("%t-", m["self"].(bool)))

	return hashcode.String(buf.String())
}

func SecGroupV2StateRefreshFunc(computeClient *gophercloud.ServiceClient, d *schema.ResourceData) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete Security Group %s.\n", d.Id())

		err := secgroups.Delete(computeClient, d.Id()).ExtractErr()
		if err != nil {
			return nil, "", err
		}

		s, err := secgroups.Get(computeClient, d.Id()).Extract()
		if err != nil {
			err = CheckDeleted(d, err, "Security Group")
			if err != nil {
				return s, "", err
			} else {
				log.Printf("[DEBUG] Successfully deleted Security Group %s", d.Id())
				return s, "DELETED", nil
			}
		}

		log.Printf("[DEBUG] Security Group %s still active.\n", d.Id())
		return s, "ACTIVE", nil
	}
}
