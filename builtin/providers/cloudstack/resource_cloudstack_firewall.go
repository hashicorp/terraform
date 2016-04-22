package cloudstack

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackFirewall() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackFirewallCreate,
		Read:   resourceCloudStackFirewallRead,
		Update: resourceCloudStackFirewallUpdate,
		Delete: resourceCloudStackFirewallDelete,

		Schema: map[string]*schema.Schema{
			"ip_address_id": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"ipaddress"},
			},

			"ipaddress": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Deprecated:    "Please use the `ip_address_id` field instead",
				ConflictsWith: []string{"ip_address_id"},
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
						"cidr_list": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"source_cidr": &schema.Schema{
							Type:       schema.TypeString,
							Optional:   true,
							Deprecated: "Please use the `cidr_list` field instead",
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
							Set:      schema.HashString,
						},

						"uuids": &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
						},
					},
				},
			},

			"parallelism": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  2,
			},
		},
	}
}

func resourceCloudStackFirewallCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Make sure all required parameters are there
	if err := verifyFirewallParams(d); err != nil {
		return err
	}

	ipaddress, ok := d.GetOk("ip_address_id")
	if !ok {
		ipaddress, ok = d.GetOk("ipaddress")
	}
	if !ok {
		return errors.New("Either `ip_address_id` or [deprecated] `ipaddress` must be provided.")
	}

	// Retrieve the ipaddress ID
	ipaddressid, e := retrieveID(cs, "ip_address", ipaddress.(string))
	if e != nil {
		return e.Error()
	}

	// We need to set this upfront in order to be able to save a partial state
	d.SetId(ipaddressid)

	// Create all rules that are configured
	if nrs := d.Get("rule").(*schema.Set); nrs.Len() > 0 {
		// Create an empty schema.Set to hold all rules
		rules := resourceCloudStackFirewall().Schema["rule"].ZeroValue().(*schema.Set)

		err := createFirewallRules(d, meta, rules, nrs)

		// We need to update this first to preserve the correct state
		d.Set("rule", rules)

		if err != nil {
			return err
		}
	}

	return resourceCloudStackFirewallRead(d, meta)
}
func createFirewallRules(
	d *schema.ResourceData,
	meta interface{},
	rules *schema.Set,
	nrs *schema.Set) error {
	var errs *multierror.Error

	var wg sync.WaitGroup
	wg.Add(nrs.Len())

	sem := make(chan struct{}, d.Get("parallelism").(int))
	for _, rule := range nrs.List() {
		// Put in a tiny sleep here to avoid DoS'ing the API
		time.Sleep(500 * time.Millisecond)

		go func(rule map[string]interface{}) {
			defer wg.Done()
			sem <- struct{}{}

			// Create a single rule
			err := createFirewallRule(d, meta, rule)

			// If we have at least one UUID, we need to save the rule
			if len(rule["uuids"].(map[string]interface{})) > 0 {
				rules.Add(rule)
			}

			if err != nil {
				errs = multierror.Append(errs, err)
			}

			<-sem
		}(rule.(map[string]interface{}))
	}

	wg.Wait()

	return errs.ErrorOrNil()
}

func createFirewallRule(
	d *schema.ResourceData,
	meta interface{},
	rule map[string]interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	uuids := rule["uuids"].(map[string]interface{})

	// Make sure all required rule parameters are there
	if err := verifyFirewallRuleParams(d, rule); err != nil {
		return err
	}

	// Create a new parameter struct
	p := cs.Firewall.NewCreateFirewallRuleParams(d.Id(), rule["protocol"].(string))

	// Set the CIDR list
	p.SetCidrlist(retrieveCidrList(rule))

	// If the protocol is ICMP set the needed ICMP parameters
	if rule["protocol"].(string) == "icmp" {
		p.SetIcmptype(rule["icmp_type"].(int))
		p.SetIcmpcode(rule["icmp_code"].(int))

		r, err := cs.Firewall.CreateFirewallRule(p)
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
			ports := &schema.Set{F: schema.HashString}

			for _, port := range ps.List() {
				if _, ok := uuids[port.(string)]; ok {
					ports.Add(port)
					rule["ports"] = ports
					continue
				}

				m := splitPorts.FindStringSubmatch(port.(string))

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

				r, err := cs.Firewall.CreateFirewallRule(p)
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

func resourceCloudStackFirewallRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get all the rules from the running environment
	p := cs.Firewall.NewListFirewallRulesParams()
	p.SetIpaddressid(d.Id())
	p.SetListall(true)

	l, err := cs.Firewall.ListFirewallRules(p)
	if err != nil {
		return err
	}

	// Make a map of all the rules so we can easily find a rule
	ruleMap := make(map[string]*cloudstack.FirewallRule, l.Count)
	for _, r := range l.FirewallRules {
		ruleMap[r.Id] = r
	}

	// Create an empty schema.Set to hold all rules
	rules := resourceCloudStackFirewall().Schema["rule"].ZeroValue().(*schema.Set)

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
				rule["protocol"] = r.Protocol
				rule["icmp_type"] = r.Icmptype
				rule["icmp_code"] = r.Icmpcode
				setCidrList(rule, r.Cidrlist)
				rules.Add(rule)
			}

			// If protocol is not ICMP, loop through all ports
			if rule["protocol"].(string) != "icmp" {
				if ps := rule["ports"].(*schema.Set); ps.Len() > 0 {

					// Create an empty schema.Set to hold all ports
					ports := &schema.Set{F: schema.HashString}

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
						rule["protocol"] = r.Protocol
						setCidrList(rule, r.Cidrlist)
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
			// We need to create and add a dummy value to a schema.Set as the
			// cidr_list is a required field and thus needs a value
			cidrs := &schema.Set{F: schema.HashString}
			cidrs.Add(uuid)

			// Make a dummy rule to hold the unknown UUID
			rule := map[string]interface{}{
				"cidr_list": cidrs,
				"protocol":  uuid,
				"uuids":     map[string]interface{}{uuid: uuid},
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

func resourceCloudStackFirewallUpdate(d *schema.ResourceData, meta interface{}) error {
	// Make sure all required parameters are there
	if err := verifyFirewallParams(d); err != nil {
		return err
	}

	// Check if the rule set as a whole has changed
	if d.HasChange("rule") {
		o, n := d.GetChange("rule")
		ors := o.(*schema.Set).Difference(n.(*schema.Set))
		nrs := n.(*schema.Set).Difference(o.(*schema.Set))

		// We need to start with a rule set containing all the rules we
		// already have and want to keep. Any rules that are not deleted
		// correctly and any newly created rules, will be added to this
		// set to make sure we end up in a consistent state
		rules := o.(*schema.Set).Intersection(n.(*schema.Set))

		// First loop through all the old rules and delete them
		if ors.Len() > 0 {
			err := deleteFirewallRules(d, meta, rules, ors)

			// We need to update this first to preserve the correct state
			d.Set("rule", rules)

			if err != nil {
				return err
			}
		}

		// Then loop through all the new rules and create them
		if nrs.Len() > 0 {
			err := createFirewallRules(d, meta, rules, nrs)

			// We need to update this first to preserve the correct state
			d.Set("rule", rules)

			if err != nil {
				return err
			}
		}
	}

	return resourceCloudStackFirewallRead(d, meta)
}

func resourceCloudStackFirewallDelete(d *schema.ResourceData, meta interface{}) error {
	// Create an empty rule set to hold all rules that where
	// not deleted correctly
	rules := resourceCloudStackFirewall().Schema["rule"].ZeroValue().(*schema.Set)

	// Delete all rules
	if ors := d.Get("rule").(*schema.Set); ors.Len() > 0 {
		err := deleteFirewallRules(d, meta, rules, ors)

		// We need to update this first to preserve the correct state
		d.Set("rule", rules)

		if err != nil {
			return err
		}
	}

	return nil
}

func deleteFirewallRules(
	d *schema.ResourceData,
	meta interface{},
	rules *schema.Set,
	ors *schema.Set) error {
	var errs *multierror.Error

	var wg sync.WaitGroup
	wg.Add(ors.Len())

	sem := make(chan struct{}, d.Get("parallelism").(int))
	for _, rule := range ors.List() {
		// Put a sleep here to avoid DoS'ing the API
		time.Sleep(500 * time.Millisecond)

		go func(rule map[string]interface{}) {
			defer wg.Done()
			sem <- struct{}{}

			// Delete a single rule
			err := deleteFirewallRule(d, meta, rule)

			// If we have at least one UUID, we need to save the rule
			if len(rule["uuids"].(map[string]interface{})) > 0 {
				rules.Add(rule)
			}

			if err != nil {
				errs = multierror.Append(errs, err)
			}

			<-sem
		}(rule.(map[string]interface{}))
	}

	wg.Wait()

	return errs.ErrorOrNil()
}

func deleteFirewallRule(
	d *schema.ResourceData,
	meta interface{},
	rule map[string]interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	uuids := rule["uuids"].(map[string]interface{})

	for k, id := range uuids {
		// We don't care about the count here, so just continue
		if k == "#" {
			continue
		}

		// Create the parameter struct
		p := cs.Firewall.NewDeleteFirewallRuleParams(id.(string))

		// Delete the rule
		if _, err := cs.Firewall.DeleteFirewallRule(p); err != nil {

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
		rule["uuids"] = uuids
	}

	return nil
}

func verifyFirewallParams(d *schema.ResourceData) error {
	managed := d.Get("managed").(bool)
	_, rules := d.GetOk("rule")

	if !rules && !managed {
		return fmt.Errorf(
			"You must supply at least one 'rule' when not using the 'managed' firewall feature")
	}

	return nil
}

func verifyFirewallRuleParams(d *schema.ResourceData, rule map[string]interface{}) error {
	cidrList := rule["cidr_list"].(*schema.Set)
	sourceCidr := rule["source_cidr"].(string)
	if cidrList.Len() == 0 && sourceCidr == "" {
		return fmt.Errorf(
			"Parameter cidr_list is a required parameter")
	}
	if cidrList.Len() > 0 && sourceCidr != "" {
		return fmt.Errorf(
			"Parameter source_cidr is deprecated and cannot be used together with cidr_list")
	}

	protocol := rule["protocol"].(string)
	if protocol != "tcp" && protocol != "udp" && protocol != "icmp" {
		return fmt.Errorf(
			"%q is not a valid protocol. Valid options are 'tcp', 'udp' and 'icmp'", protocol)
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
		if ports, ok := rule["ports"].(*schema.Set); ok {
			for _, port := range ports.List() {
				m := splitPorts.FindStringSubmatch(port.(string))
				if m == nil {
					return fmt.Errorf(
						"%q is not a valid port value. Valid options are '80' or '80-90'", port.(string))
				}
			}
		} else {
			return fmt.Errorf(
				"Parameter ports is a required parameter when *not* using protocol 'icmp'")
		}
	}

	return nil
}
