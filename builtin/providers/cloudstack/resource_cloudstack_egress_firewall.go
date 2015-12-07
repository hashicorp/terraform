package cloudstack

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
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
	if nrs := d.Get("rule").(*schema.Set); nrs.Len() > 0 {
		// Create an empty schema.Set to hold all rules
		rules := resourceCloudStackEgressFirewall().Schema["rule"].ZeroValue().(*schema.Set)

		err := createEgressFirewallRules(d, meta, rules, nrs)

		// We need to update this first to preserve the correct state
		d.Set("rule", rules)

		if err != nil {
			return err
		}
	}

	return resourceCloudStackEgressFirewallRead(d, meta)
}

func createEgressFirewallRules(
	d *schema.ResourceData,
	meta interface{},
	rules *schema.Set,
	nrs *schema.Set) error {
	var errs *multierror.Error

	var wg sync.WaitGroup
	wg.Add(nrs.Len())

	sem := make(chan struct{}, 10)
	for _, rule := range nrs.List() {
		// Put in a tiny sleep here to avoid DoS'ing the API
		time.Sleep(500 * time.Millisecond)

		go func(rule map[string]interface{}) {
			defer wg.Done()
			sem <- struct{}{}

			// Create a single rule
			err := createEgressFirewallRule(d, meta, rule)

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
func createEgressFirewallRule(
	d *schema.ResourceData,
	meta interface{},
	rule map[string]interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	uuids := rule["uuids"].(map[string]interface{})

	// Make sure all required rule parameters are there
	if err := verifyEgressFirewallRuleParams(d, rule); err != nil {
		return err
	}

	// Create a new parameter struct
	p := cs.Firewall.NewCreateEgressFirewallRuleParams(d.Id(), rule["protocol"].(string))

	// Set the CIDR list
	p.SetCidrlist(retrieveCidrList(rule))

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
			ports := &schema.Set{F: schema.HashString}

			// Define a regexp for parsing the port
			re := regexp.MustCompile(`^(\d+)(?:-(\d+))?$`)

			for _, port := range ps.List() {
				if _, ok := uuids[port.(string)]; ok {
					ports.Add(port)
					rule["ports"] = ports
					continue
				}

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
	rules := resourceCloudStackEgressFirewall().Schema["rule"].ZeroValue().(*schema.Set)

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
				"cidr_list": uuid,
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

		// We need to start with a rule set containing all the rules we
		// already have and want to keep. Any rules that are not deleted
		// correctly and any newly created rules, will be added to this
		// set to make sure we end up in a consistent state
		rules := o.(*schema.Set).Intersection(n.(*schema.Set))

		// First loop through all the old rules and delete them
		if ors.Len() > 0 {
			err := deleteEgressFirewallRules(d, meta, rules, ors)

			// We need to update this first to preserve the correct state
			d.Set("rule", rules)

			if err != nil {
				return err
			}
		}

		// Then loop through all the new rules and create them
		if nrs.Len() > 0 {
			err := createEgressFirewallRules(d, meta, rules, nrs)

			// We need to update this first to preserve the correct state
			d.Set("rule", rules)

			if err != nil {
				return err
			}
		}
	}

	return resourceCloudStackEgressFirewallRead(d, meta)
}

func resourceCloudStackEgressFirewallDelete(d *schema.ResourceData, meta interface{}) error {
	// Create an empty rule set to hold all rules that where
	// not deleted correctly
	rules := resourceCloudStackEgressFirewall().Schema["rule"].ZeroValue().(*schema.Set)

	// Delete all rules
	if ors := d.Get("rule").(*schema.Set); ors.Len() > 0 {
		err := deleteEgressFirewallRules(d, meta, rules, ors)

		// We need to update this first to preserve the correct state
		d.Set("rule", rules)

		if err != nil {
			return err
		}
	}

	return nil
}

func deleteEgressFirewallRules(
	d *schema.ResourceData,
	meta interface{},
	rules *schema.Set,
	ors *schema.Set) error {
	var errs *multierror.Error

	var wg sync.WaitGroup
	wg.Add(ors.Len())

	sem := make(chan struct{}, 10)
	for _, rule := range ors.List() {
		// Put a sleep here to avoid DoS'ing the API
		time.Sleep(500 * time.Millisecond)

		go func(rule map[string]interface{}) {
			defer wg.Done()
			sem <- struct{}{}

			// Delete a single rule
			err := deleteEgressFirewallRule(d, meta, rule)

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

func deleteEgressFirewallRule(
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
		rule["uuids"] = uuids
	}

	return nil
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
