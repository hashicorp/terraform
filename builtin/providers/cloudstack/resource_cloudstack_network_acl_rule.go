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

func resourceCloudStackNetworkACLRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackNetworkACLRuleCreate,
		Read:   resourceCloudStackNetworkACLRuleRead,
		Update: resourceCloudStackNetworkACLRuleUpdate,
		Delete: resourceCloudStackNetworkACLRuleDelete,

		Schema: map[string]*schema.Schema{
			"aclid": &schema.Schema{
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
						"action": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "allow",
						},

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

						"traffic_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "ingress",
						},

						"uuids": &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
						},
					},
				},
				Set: resourceCloudStackNetworkACLRuleHash,
			},
		},
	}
}

func resourceCloudStackNetworkACLRuleCreate(d *schema.ResourceData, meta interface{}) error {
	// Make sure all required parameters are there
	if err := verifyNetworkACLParams(d); err != nil {
		return err
	}

	// We need to set this upfront in order to be able to save a partial state
	d.SetId(d.Get("aclid").(string))

	// Create all rules that are configured
	if rs := d.Get("rule").(*schema.Set); rs.Len() > 0 {

		// Create an empty schema.Set to hold all rules
		rules := &schema.Set{
			F: resourceCloudStackNetworkACLRuleHash,
		}

		for _, rule := range rs.List() {
			// Create a single rule
			err := resourceCloudStackNetworkACLRuleCreateRule(d, meta, rule.(map[string]interface{}))

			// We need to update this first to preserve the correct state
			rules.Add(rule)
			d.Set("rule", rules)

			if err != nil {
				return err
			}
		}
	}

	return resourceCloudStackNetworkACLRuleRead(d, meta)
}

func resourceCloudStackNetworkACLRuleCreateRule(
	d *schema.ResourceData, meta interface{}, rule map[string]interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	uuids := rule["uuids"].(map[string]interface{})

	// Make sure all required parameters are there
	if err := verifyNetworkACLRuleParams(d, rule); err != nil {
		return err
	}

	// Create a new parameter struct
	p := cs.NetworkACL.NewCreateNetworkACLParams(rule["protocol"].(string))

	// Set the acl ID
	p.SetAclid(d.Id())

	// Set the action
	p.SetAction(rule["action"].(string))

	// Set the CIDR list
	p.SetCidrlist([]string{rule["source_cidr"].(string)})

	// Set the traffic type
	p.SetTraffictype(rule["traffic_type"].(string))

	// If the protocol is ICMP set the needed ICMP parameters
	if rule["protocol"].(string) == "icmp" {
		p.SetIcmptype(rule["icmp_type"].(int))
		p.SetIcmpcode(rule["icmp_code"].(int))

		r, err := Retry(4, retryableACLCreationFunc(cs, p))
		if err != nil {
			return err
		}

		uuids["icmp"] = r.(*cloudstack.CreateNetworkACLResponse).Id
		rule["uuids"] = uuids
	}

	// If the protocol is ALL set the needed parameters
	if rule["protocol"].(string) == "all" {
		r, err := Retry(4, retryableACLCreationFunc(cs, p))
		if err != nil {
			return err
		}

		uuids["all"] = r.(*cloudstack.CreateNetworkACLResponse).Id
		rule["uuids"] = uuids
	}

	// If protocol is TCP or UDP, loop through all ports
	if rule["protocol"].(string) == "tcp" || rule["protocol"].(string) == "udp" {
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

				r, err := Retry(4, retryableACLCreationFunc(cs, p))
				if err != nil {
					return err
				}

				ports.Add(port)
				rule["ports"] = ports

				uuids[port.(string)] = r.(*cloudstack.CreateNetworkACLResponse).Id
				rule["uuids"] = uuids
			}
		}
	}

	return nil
}

func resourceCloudStackNetworkACLRuleRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create an empty schema.Set to hold all rules
	rules := &schema.Set{
		F: resourceCloudStackNetworkACLRuleHash,
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
				r, count, err := cs.NetworkACL.GetNetworkACLByID(id.(string))
				// If the count == 0, there is no object found for this UUID
				if err != nil {
					if count == 0 {
						delete(uuids, "icmp")
						continue
					}

					return err
				}

				// Update the values
				rule["action"] = strings.ToLower(r.Action)
				rule["source_cidr"] = r.Cidrlist
				rule["protocol"] = r.Protocol
				rule["icmp_type"] = r.Icmptype
				rule["icmp_code"] = r.Icmpcode
				rule["traffic_type"] = strings.ToLower(r.Traffictype)
				rules.Add(rule)
			}

			if rule["protocol"].(string) == "all" {
				id, ok := uuids["all"]
				if !ok {
					continue
				}

				// Get the rule
				r, count, err := cs.NetworkACL.GetNetworkACLByID(id.(string))
				// If the count == 0, there is no object found for this UUID
				if err != nil {
					if count == 0 {
						delete(uuids, "all")
						continue
					}

					return err
				}

				// Update the values
				rule["action"] = strings.ToLower(r.Action)
				rule["source_cidr"] = r.Cidrlist
				rule["protocol"] = r.Protocol
				rule["traffic_type"] = strings.ToLower(r.Traffictype)
				rules.Add(rule)
			}

			// If protocol is tcp or udp, loop through all ports
			if rule["protocol"].(string) == "tcp" || rule["protocol"].(string) == "udp" {
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
						r, count, err := cs.NetworkACL.GetNetworkACLByID(id.(string))
						if err != nil {
							if count == 0 {
								delete(uuids, port.(string))
								continue
							}

							return err
						}

						// Update the values
						rule["action"] = strings.ToLower(r.Action)
						rule["source_cidr"] = r.Cidrlist
						rule["protocol"] = r.Protocol
						rule["traffic_type"] = strings.ToLower(r.Traffictype)
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
	if managed {
		// Get all the rules from the running environment
		p := cs.NetworkACL.NewListNetworkACLsParams()
		p.SetAclid(d.Id())
		p.SetListall(true)

		r, err := cs.NetworkACL.ListNetworkACLs(p)
		if err != nil {
			return err
		}

		// Add all UUIDs to the uuids map
		uuids := make(map[string]interface{}, len(r.NetworkACLs))
		for _, r := range r.NetworkACLs {
			uuids[r.Id] = r.Id
		}

		// Delete all expected UUIDs from the uuids map
		for _, rule := range rules.List() {
			rule := rule.(map[string]interface{})

			for _, id := range rule["uuids"].(map[string]interface{}) {
				delete(uuids, id.(string))
			}
		}

		if len(uuids) > 0 {
			// Make a dummy rule to hold all unknown UUIDs
			rule := map[string]interface{}{
				"source_cidr": "N/A",
				"protocol":    "N/A",
				"uuids":       uuids,
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

func resourceCloudStackNetworkACLRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	// Make sure all required parameters are there
	if err := verifyNetworkACLParams(d); err != nil {
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
			err := resourceCloudStackNetworkACLRuleDeleteRule(d, meta, rule.(map[string]interface{}))
			if err != nil {
				return err
			}
		}

		// Make sure we save the state of the currently configured rules
		rules := o.(*schema.Set).Intersection(n.(*schema.Set))
		d.Set("rule", rules)

		// Then loop through all the currently configured rules and create the new ones
		for _, rule := range nrs.List() {
			// When succesfully deleted, re-create it again if it still exists
			err := resourceCloudStackNetworkACLRuleCreateRule(d, meta, rule.(map[string]interface{}))

			// We need to update this first to preserve the correct state
			rules.Add(rule)
			d.Set("rule", rules)

			if err != nil {
				return err
			}
		}
	}

	return resourceCloudStackNetworkACLRuleRead(d, meta)
}

func resourceCloudStackNetworkACLRuleDelete(d *schema.ResourceData, meta interface{}) error {
	// Delete all rules
	if rs := d.Get("rule").(*schema.Set); rs.Len() > 0 {
		for _, rule := range rs.List() {
			// Delete a single rule
			err := resourceCloudStackNetworkACLRuleDeleteRule(d, meta, rule.(map[string]interface{}))

			// We need to update this first to preserve the correct state
			d.Set("rule", rs)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func resourceCloudStackNetworkACLRuleDeleteRule(
	d *schema.ResourceData, meta interface{}, rule map[string]interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	uuids := rule["uuids"].(map[string]interface{})

	for k, id := range uuids {
		// We don't care about the count here, so just continue
		if k == "#" {
			continue
		}

		// Create the parameter struct
		p := cs.NetworkACL.NewDeleteNetworkACLParams(id.(string))

		// Delete the rule
		if _, err := cs.NetworkACL.DeleteNetworkACL(p); err != nil {

			// This is a very poor way to be told the UUID does no longer exist :(
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

func resourceCloudStackNetworkACLRuleHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	// This is a little ugly, but it's needed because these arguments have
	// a default value that needs to be part of the string to hash
	var action, trafficType string
	if a, ok := m["action"]; ok {
		action = a.(string)
	} else {
		action = "allow"
	}
	if t, ok := m["traffic_type"]; ok {
		trafficType = t.(string)
	} else {
		trafficType = "ingress"
	}

	buf.WriteString(fmt.Sprintf(
		"%s-%s-%s-%s-",
		action,
		m["source_cidr"].(string),
		m["protocol"].(string),
		trafficType))

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

func verifyNetworkACLParams(d *schema.ResourceData) error {
	managed := d.Get("managed").(bool)
	_, rules := d.GetOk("rule")

	if !rules && !managed {
		return fmt.Errorf(
			"You must supply at least one 'rule' when not using the 'managed' firewall feature")
	}

	return nil
}

func verifyNetworkACLRuleParams(d *schema.ResourceData, rule map[string]interface{}) error {
	action := rule["action"].(string)
	if action != "allow" && action != "deny" {
		return fmt.Errorf("Parameter action only accepts 'allow' or 'deny' as values")
	}

	protocol := rule["protocol"].(string)
	switch protocol {
	case "icmp":
		if _, ok := rule["icmp_type"]; !ok {
			return fmt.Errorf(
				"Parameter icmp_type is a required parameter when using protocol 'icmp'")
		}
		if _, ok := rule["icmp_code"]; !ok {
			return fmt.Errorf(
				"Parameter icmp_code is a required parameter when using protocol 'icmp'")
		}
	case "all":
		// No additional test are needed, so just leave this empty...
	case "tcp", "udp":
		if _, ok := rule["ports"]; !ok {
			return fmt.Errorf(
				"Parameter ports is a required parameter when *not* using protocol 'icmp'")
		}
	default:
		_, err := strconv.ParseInt(protocol, 0, 0)
		if err != nil {
			return fmt.Errorf(
				"%s is not a valid protocol. Valid options are 'tcp', 'udp', "+
					"'icmp', 'all' or a valid protocol number", protocol)
		}
	}

	traffic := rule["traffic_type"].(string)
	if traffic != "ingress" && traffic != "egress" {
		return fmt.Errorf(
			"Parameter traffic_type only accepts 'ingress' or 'egress' as values")
	}

	return nil
}

func retryableACLCreationFunc(
	cs *cloudstack.CloudStackClient,
	p *cloudstack.CreateNetworkACLParams) func() (interface{}, error) {
	return func() (interface{}, error) {
		r, err := cs.NetworkACL.CreateNetworkACL(p)
		if err != nil {
			return nil, err
		}
		return r, nil
	}
}
