package azure

import (
	"bytes"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/svanharmelen/azure-sdk-for-go/management"
	"github.com/svanharmelen/azure-sdk-for-go/management/networksecuritygroup"
)

func resourceAzureSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureSecurityGroupCreate,
		Read:   resourceAzureSecurityGroupRead,
		Update: resourceAzureSecurityGroupUpdate,
		Delete: resourceAzureSecurityGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"label": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"rule": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "Inbound",
						},

						"priority": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"action": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "Allow",
						},

						"source_cidr": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"source_port": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"destination_cidr": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"destination_port": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "TCP",
						},
					},
				},
				Set: resourceAzureSecurityGroupRuleHash,
			},
		},
	}
}

func resourceAzureSecurityGroupCreate(d *schema.ResourceData, meta interface{}) (err error) {
	mc := meta.(*Client).mgmtClient

	name := d.Get("name").(string)

	// Compute/set the label
	label := d.Get("label").(string)
	if label == "" {
		label = name
	}

	req, err := networksecuritygroup.NewClient(mc).CreateNetworkSecurityGroup(
		name,
		label,
		d.Get("location").(string),
	)
	if err != nil {
		return fmt.Errorf("Error creating Network Security Group %s: %s", name, err)
	}

	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf(
			"Error waiting for Network Security Group %s to be created: %s", name, err)
	}

	d.SetId(name)

	// Create all rules that are configured
	if rs := d.Get("rule").(*schema.Set); rs.Len() > 0 {

		// Create an empty schema.Set to hold all rules
		rules := &schema.Set{
			F: resourceAzureSecurityGroupRuleHash,
		}

		for _, rule := range rs.List() {
			// Create a single rule
			err := resourceAzureSecurityGroupRuleCreate(d, meta, rule.(map[string]interface{}))

			// We need to update this first to preserve the correct state
			rules.Add(rule)
			d.Set("rule", rules)

			if err != nil {
				return err
			}
		}
	}

	return resourceAzureSecurityGroupRead(d, meta)
}

func resourceAzureSecurityGroupRuleCreate(
	d *schema.ResourceData,
	meta interface{},
	rule map[string]interface{}) error {
	mc := meta.(*Client).mgmtClient

	// Make sure all required parameters are there
	if err := verifySecurityGroupRuleParams(rule); err != nil {
		return err
	}

	name := rule["name"].(string)

	// Create the rule
	req, err := networksecuritygroup.NewClient(mc).SetNetworkSecurityGroupRule(d.Id(),
		networksecuritygroup.RuleRequest{
			Name:                     name,
			Type:                     networksecuritygroup.RuleType(rule["type"].(string)),
			Priority:                 rule["priority"].(int),
			Action:                   networksecuritygroup.RuleAction(rule["action"].(string)),
			SourceAddressPrefix:      rule["source_cidr"].(string),
			SourcePortRange:          rule["source_port"].(string),
			DestinationAddressPrefix: rule["destination_cidr"].(string),
			DestinationPortRange:     rule["destination_port"].(string),
			Protocol:                 networksecuritygroup.RuleProtocol(rule["protocol"].(string)),
		},
	)
	if err != nil {
		return fmt.Errorf("Error creating Network Security Group rule %s: %s", name, err)
	}

	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf(
			"Error waiting for Network Security Group rule %s to be created: %s", name, err)
	}

	return nil
}

func resourceAzureSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*Client).mgmtClient

	sg, err := networksecuritygroup.NewClient(mc).GetNetworkSecurityGroup(d.Id())
	if err != nil {
		if management.IsResourceNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving Network Security Group %s: %s", d.Id(), err)
	}

	d.Set("label", sg.Label)
	d.Set("location", sg.Location)

	// Create an empty schema.Set to hold all rules
	rules := &schema.Set{
		F: resourceAzureSecurityGroupRuleHash,
	}

	for _, r := range sg.Rules {
		if !r.IsDefault {
			rule := map[string]interface{}{
				"name":             r.Name,
				"type":             string(r.Type),
				"priority":         r.Priority,
				"action":           string(r.Action),
				"source_cidr":      r.SourceAddressPrefix,
				"source_port":      r.SourcePortRange,
				"destination_cidr": r.DestinationAddressPrefix,
				"destination_port": r.DestinationPortRange,
				"protocol":         string(r.Protocol),
			}
			rules.Add(rule)
		}
	}

	d.Set("rule", rules)

	return nil
}

func resourceAzureSecurityGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	// Check if the rule set as a whole has changed
	if d.HasChange("rule") {
		o, n := d.GetChange("rule")
		ors := o.(*schema.Set).Difference(n.(*schema.Set))
		nrs := n.(*schema.Set).Difference(o.(*schema.Set))

		// Now first loop through all the old rules and delete any obsolete ones
		for _, rule := range ors.List() {
			// Delete the rule as it no longer exists in the config
			err := resourceAzureSecurityGroupRuleDelete(d, meta, rule.(map[string]interface{}))
			if err != nil {
				return err
			}
		}

		// Make sure we save the state of the currently configured rules
		rules := o.(*schema.Set).Intersection(n.(*schema.Set))
		d.Set("rule", rules)

		// Then loop through al the currently configured rules and create the new ones
		for _, rule := range nrs.List() {
			err := resourceAzureSecurityGroupRuleCreate(d, meta, rule.(map[string]interface{}))

			// We need to update this first to preserve the correct state
			rules.Add(rule)
			d.Set("rule", rules)

			if err != nil {
				return err
			}
		}
	}

	return resourceAzureSecurityGroupRead(d, meta)
}

func resourceAzureSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*Client).mgmtClient

	log.Printf("[DEBUG] Deleting Network Security Group: %s", d.Id())
	req, err := networksecuritygroup.NewClient(mc).DeleteNetworkSecurityGroup(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting Network Security Group %s: %s", d.Id(), err)
	}

	// Wait until the network security group is deleted
	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf(
			"Error waiting for Network Security Group %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}

func resourceAzureSecurityGroupRuleDelete(
	d *schema.ResourceData, meta interface{}, rule map[string]interface{}) error {
	mc := meta.(*Client).mgmtClient

	name := rule["name"].(string)

	// Delete the rule
	req, err := networksecuritygroup.NewClient(mc).DeleteNetworkSecurityGroupRule(d.Id(), name)
	if err != nil {
		if management.IsResourceNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("Error deleting Network Security Group rule %s: %s", name, err)
	}

	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf(
			"Error waiting for Network Security Group rule %s to be deleted: %s", name, err)
	}

	return nil
}

func resourceAzureSecurityGroupRuleHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf(
		"%s-%d-%s-%s-%s-%s-%s-%s",
		m["type"].(string),
		m["priority"].(int),
		m["action"].(string),
		m["source_cidr"].(string),
		m["source_port"].(string),
		m["destination_cidr"].(string),
		m["destination_port"].(string),
		m["protocol"].(string)))

	return hashcode.String(buf.String())
}

func verifySecurityGroupRuleParams(rule map[string]interface{}) error {
	typ := rule["type"].(string)
	if typ != "Inbound" && typ != "Outbound" {
		return fmt.Errorf("Parameter type only accepts 'Inbound' or 'Outbound' as values")
	}

	action := rule["action"].(string)
	if action != "Allow" && action != "Deny" {
		return fmt.Errorf("Parameter action only accepts 'Allow' or 'Deny' as values")
	}

	protocol := rule["protocol"].(string)
	if protocol != "TCP" && protocol != "UDP" && protocol != "*" {
		_, err := strconv.ParseInt(protocol, 0, 0)
		if err != nil {
			return fmt.Errorf(
				"Parameter type only accepts 'TCP', 'UDP' or '*' as values")
		}
	}

	return nil
}
