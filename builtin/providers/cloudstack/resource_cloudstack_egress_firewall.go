package cloudstack

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackEgressFirewall() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackEgressFirewallCreate,
		Read:   resourceCloudStackEgressFirewallRead,
		Update: resourceCloudStackEgressFirewallUpdate,
		Delete: resourceCloudStackEgressFirewallDelete,

		Schema: map[string]*schema.Schema{
			"network": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"managed": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"rule": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source_cidr": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"icmp_type": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},

						"icmp_code": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},

						"ports": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set: func(v interface{}) int {
								return hashcode.String(v.(string))
							},
						},

						"uuids": &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
						},
					},
				},
				Set: resourceCloudStackEgressFirewallRuleHash,
			},
		},
	}
}

func resourceCloudStackEgressFirewallCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Make sure all required parameters are there
	if err := verifyEgressFirewallParams(d); err != nil {
		return err
	}

	// Retrieve the network ID
	networkid, e := retrieveID(cs, "network", d.Get("network").(string))
	if e != nil {
		return e.Error()
	}

	// We need to set this upfront in order to be able to save a partial state
	d.SetId(networkid)

	// Create all rules that are configured
	if rs := d.Get("rule").(*schema.Set); rs.Len() > 0 {

		// Create an empty schema.Set to hold all rules
		rules := &schema.Set{
			F: resourceCloudStackEgressFirewallRuleHash,
		}

		for _, rule := range rs.List() {
			// Create a single rule
			err := resourceCloudStackEgressFirewallCreateRule(d, meta, rule.(map[string]interface{}))

			// We need to update this first to preserve the correct state
			rules.Add(rule)
			d.Set("rule", rules)

			if err != nil {
				return err
			}
		}
	}

	return resourceCloudStackEgressFirewallRead(d, meta)
}

func resourceCloudStackEgressFirewallCreateRule(
	d *schema.ResourceData, meta interface{}, rule map[string]interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	uuids := rule["uuids"].(map[string]interface{})

	// Make sure all required rule parameters are there
	if err := verifyEgressFirewallRuleParams(d, rule); err != nil {
		return err
	}

	// Create a new parameter struct
	p := cs.Firewall.NewCreateEgressFirewallRuleParams(d.Id(), rule["protocol"].(string))

	// Set the CIDR list
	p.SetCidrlist([]string{rule["source_cidr"].(string)})

	// If the protocol is ICMP set the needed ICMP parameters
	if rule["protocol"].(string) == "icmp" {
		p.SetIcmptype(rule["icmp_type"].(int))
		p.SetIcmpcode(rule["icmp_code"].(int))

		r, err := cs.Firewall.CreateEgressFirewallRule(p)
		if err != nil {
			return err
		}
		uuids["icmp"] = r.Id
		rule["uuids"] = uuids
	}

	// If protocol is not ICMP, loop through all ports
	if rule["protocol"].(string) != "icmp" {
		if ps := rule["ports"].(*schema.Set); ps.Len() > 0 {

			// Create an empty schema.Set to hold all processed ports
			ports := &schema.Set{
				F: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			}

			for _, port := range ps.List() {
				re := regexp.MustCompile(`^(\d+)(?:-(\d+))?$`)
				m := re.FindStringSubmatch(port.(string))

				startPort, err := strconv.Atoi(m[1])
				if err != nil {
					return err
				}

				endPort := startPort
				if m[2] != "" {
					endPort, err = strconv.Atoi(m[2])
					if err != nil {
						return err
					}
				}

				p.SetStartport(startPort)
				p.SetEndport(endPort)

				r, err := cs.Firewall.CreateEgressFirewallRule(p)
				if err != nil {
					return err
				}

				ports.Add(port)
				rule["ports"] = ports

				uuids[port.(string)] = r.Id
				rule["uuids"] = uuids
			}
		}
	}

	return nil
}

func resourceCloudStackEgressFirewallRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get all the rules from the running environment
	p := cs.Firewall.NewListEgressFirewallRulesParams()
	p.SetNetworkid(d.Id())
	p.SetListall(true)

	l, err := cs.Firewall.ListEgressFirewallRules(p)
	if err != nil {
		return err
	}

	// Make a map of all the rules so we can easily find a rule
	ruleMap := make(map[string]*cloudstack.EgressFirewallRule, l.Count)
	for _, r := range l.EgressFirewallRules {
		ruleMap[r.Id] = r
	}

	// Create an empty schema.Set to hold all rules
	rules := &schema.Set{
		F: resourceCloudStackEgressFirewallRuleHash,
	}

	// Read all rules that are configured
	if rs := d.Get("rule").(*schema.Set); rs.Len() > 0 {
		for _, rule := range rs.List() {
			rule := rule.(map[string]interface{})
			uuids := rule["uuids"].(map[string]interface{})

			if rule["protocol"].(string) == "icmp" {
				id, ok := uuids["icmp"]
				if !ok {
					continue
				}

				// Get the rule
				r, ok := ruleMap[id.(string)]
				if !ok {
					delete(uuids, "icmp")
					continue
				}

				// Delete the known rule so only unknown rules remain in the ruleMap
				delete(ruleMap, id.(string))

				// Update the values
				rule["source_cidr"] = r.Cidrlist
				rule["protocol"] = r.Protocol
				rule["icmp_type"] = r.Icmptype
				rule["icmp_code"] = r.Icmpcode
				rules.Add(rule)
			}

			// If protocol is not ICMP, loop through all ports
			if rule["protocol"].(string) != "icmp" {
				if ps := rule["ports"].(*schema.Set); ps.Len() > 0 {

					// Create an empty schema.Set to hold all ports
					ports := &schema.Set{
						F: func(v interface{}) int {
							return hashcode.String(v.(string))
						},
					}

					// Loop through all ports and retrieve their info
					for _, port := range ps.List() {
						id, ok := uuids[port.(string)]
						if !ok {
							continue
						}

						// Get the rule
						r, ok := ruleMap[id.(string)]
						if !ok {
							delete(uuids, port.(string))
							continue
						}

						// Delete the known rule so only unknown rules remain in the ruleMap
						delete(ruleMap, id.(string))

						// Update the values
						rule["source_cidr"] = r.Cidrlist
						rule["protocol"] = r.Protocol
						ports.Add(port)
					}

					// If there is at least one port found, add this rule to the rules set
					if ports.Len() > 0 {
						rule["ports"] = ports
						rules.Add(rule)
					}
				}
			}
		}
	}

	// If this is a managed firewall, add all unknown rules into a single dummy rule
	managed := d.Get("managed").(bool)
	if managed && len(ruleMap) > 0 {
		for uuid := range ruleMap {
			// Make a dummy rule to hold the unknown UUID
			rule := map[string]interface{}{
				"source_cidr": uuid,
				"protocol":    uuid,
				"uuids":       map[string]interface{}{uuid: uuid},
			}

			// Add the dummy rule to the rules set
			rules.Add(rule)
		}
	}

	if rules.Len() > 0 {
		d.Set("rule", rules)
	} else if !managed {
		d.SetId("")
	}

	return nil
}

func resourceCloudStackEgressFirewallUpdate(d *schema.ResourceData, meta interface{}) error {
	// Make sure all required parameters are there
	if err := verifyEgressFirewallParams(d); err != nil {
		return err
	}

	// Check if the rule set as a whole has changed
	if d.HasChange("rule") {
		o, n := d.GetChange("rule")
		ors := o.(*schema.Set).Difference(n.(*schema.Set))
		nrs := n.(*schema.Set).Difference(o.(*schema.Set))

		// Now first loop through all the old rules and delete any obsolete ones
		for _, rule := range ors.List() {
			// Delete the rule as it no longer exists in the config
			err := resourceCloudStackEgressFirewallDeleteRule(d, meta, rule.(map[string]interface{}))
			if err != nil {
				return err
			}
		}

		// Make sure we save the state of the currently configured rules
		rules := o.(*schema.Set).Intersection(n.(*schema.Set))
		d.Set("rule", rules)

		// Then loop through all the currently configured rules and create the new ones
		for _, rule := range nrs.List() {
			// When successfully deleted, re-create it again if it still exists
			err := resourceCloudStackEgressFirewallCreateRule(
				d, meta, rule.(map[string]interface{}))

			// We need to update this first to preserve the correct state
			rules.Add(rule)
			d.Set("rule", rules)

			if err != nil {
				return err
			}
		}
	}

	return resourceCloudStackEgressFirewallRead(d, meta)
}

func resourceCloudStackEgressFirewallDelete(d *schema.ResourceData, meta interface{}) error {
	// Delete all rules
	if rs := d.Get("rule").(*schema.Set); rs.Len() > 0 {
		for _, rule := range rs.List() {
			// Delete a single rule
			err := resourceCloudStackEgressFirewallDeleteRule(d, meta, rule.(map[string]interface{}))

			// We need to update this first to preserve the correct state
			d.Set("rule", rs)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func resourceCloudStackEgressFirewallDeleteRule(
	d *schema.ResourceData, meta interface{}, rule map[string]interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	uuids := rule["uuids"].(map[string]interface{})

	for k, id := range uuids {
		// We don't care about the count here, so just continue
		if k == "#" {
			continue
		}

		// Create the parameter struct
		p := cs.Firewall.NewDeleteEgressFirewallRuleParams(id.(string))

		// Delete the rule
		if _, err := cs.Firewall.DeleteEgressFirewallRule(p); err != nil {

			// This is a very poor way to be told the ID does no longer exist :(
			if strings.Contains(err.Error(), fmt.Sprintf(
				"Invalid parameter id value=%s due to incorrect long value format, "+
					"or entity does not exist", id.(string))) {
				delete(uuids, k)
				continue
			}

			return err
		}

		// Delete the UUID of this rule
		delete(uuids, k)
	}

	// Update the UUIDs
	rule["uuids"] = uuids

	return nil
}

func resourceCloudStackEgressFirewallRuleHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf(
		"%s-%s-", m["source_cidr"].(string), m["protocol"].(string)))

	if v, ok := m["icmp_type"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}

	if v, ok := m["icmp_code"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}

	// We need to make sure to sort the strings below so that we always
	// generate the same hash code no matter what is in the set.
	if v, ok := m["ports"]; ok {
		vs := v.(*schema.Set).List()
		s := make([]string, len(vs))

		for i, raw := range vs {
			s[i] = raw.(string)
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}

	return hashcode.String(buf.String())
}

func verifyEgressFirewallParams(d *schema.ResourceData) error {
	managed := d.Get("managed").(bool)
	_, rules := d.GetOk("rule")

	if !rules && !managed {
		return fmt.Errorf(
			"You must supply at least one 'rule' when not using the 'managed' firewall feature")
	}

	return nil
}

func verifyEgressFirewallRuleParams(d *schema.ResourceData, rule map[string]interface{}) error {
	protocol := rule["protocol"].(string)
	if protocol != "tcp" && protocol != "udp" && protocol != "icmp" {
		return fmt.Errorf(
			"%s is not a valid protocol. Valid options are 'tcp', 'udp' and 'icmp'", protocol)
	}

	if protocol == "icmp" {
		if _, ok := rule["icmp_type"]; !ok {
			return fmt.Errorf(
				"Parameter icmp_type is a required parameter when using protocol 'icmp'")
		}
		if _, ok := rule["icmp_code"]; !ok {
			return fmt.Errorf(
				"Parameter icmp_code is a required parameter when using protocol 'icmp'")
		}
	} else {
		if _, ok := rule["ports"]; !ok {
			return fmt.Errorf(
				"Parameter port is a required parameter when using protocol 'tcp' or 'udp'")
		}
	}

	return nil
}
