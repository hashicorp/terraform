package cloudstack

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

type authorizeSecurityGroupParams interface {
	SetCidrlist([]string)
	SetIcmptype(int)
	SetIcmpcode(int)
	SetStartport(int)
	SetEndport(int)
	SetProtocol(string)
	SetSecuritygroupid(string)
	SetUsersecuritygrouplist(map[string]string)
}

func resourceCloudStackSecurityGroupRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackSecurityGroupRuleCreate,
		Read:   resourceCloudStackSecurityGroupRuleRead,
		Update: resourceCloudStackSecurityGroupRuleUpdate,
		Delete: resourceCloudStackSecurityGroupRuleDelete,

		Schema: map[string]*schema.Schema{
			"security_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"rule": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cidr_list": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
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

						"traffic_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "ingress",
						},

						"user_security_group_list": &schema.Schema{
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

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"parallelism": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  2,
			},
		},
	}
}

func resourceCloudStackSecurityGroupRuleCreate(d *schema.ResourceData, meta interface{}) error {
	// We need to set this upfront in order to be able to save a partial state
	d.SetId(d.Get("security_group_id").(string))

	// Create all rules that are configured
	if nrs := d.Get("rule").(*schema.Set); nrs.Len() > 0 {
		// Create an empty rule set to hold all newly created rules
		rules := resourceCloudStackSecurityGroupRule().Schema["rule"].ZeroValue().(*schema.Set)

		err := createSecurityGroupRules(d, meta, rules, nrs)

		// We need to update this first to preserve the correct state
		d.Set("rule", rules)

		if err != nil {
			return err
		}
	}

	return resourceCloudStackSecurityGroupRuleRead(d, meta)
}

func createSecurityGroupRules(d *schema.ResourceData, meta interface{}, rules *schema.Set, nrs *schema.Set) error {
	cs := meta.(*cloudstack.CloudStackClient)
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

			// Make sure all required parameters are there
			if err := verifySecurityGroupRuleParams(d, rule); err != nil {
				errs = multierror.Append(errs, err)
				return
			}

			var p authorizeSecurityGroupParams

			if cidrList, ok := rule["cidr_list"].(*schema.Set); ok && cidrList.Len() > 0 {
				for _, cidr := range cidrList.List() {
					// Create a new parameter struct
					switch rule["traffic_type"].(string) {
					case "ingress":
						p = cs.SecurityGroup.NewAuthorizeSecurityGroupIngressParams()
					case "egress":
						p = cs.SecurityGroup.NewAuthorizeSecurityGroupEgressParams()
					}

					p.SetSecuritygroupid(d.Id())
					p.SetCidrlist([]string{cidr.(string)})

					// Create a single rule
					err := createSecurityGroupRule(d, meta, rule, p, cidr.(string))
					if err != nil {
						errs = multierror.Append(errs, err)
					}
				}
			}

			if usgList, ok := rule["user_security_group_list"].(*schema.Set); ok && usgList.Len() > 0 {
				for _, usg := range usgList.List() {
					sg, _, err := cs.SecurityGroup.GetSecurityGroupByName(
						usg.(string),
						cloudstack.WithProject(d.Get("project").(string)),
					)
					if err != nil {
						errs = multierror.Append(errs, err)
						continue
					}

					// Create a new parameter struct
					switch rule["traffic_type"].(string) {
					case "ingress":
						p = cs.SecurityGroup.NewAuthorizeSecurityGroupIngressParams()
					case "egress":
						p = cs.SecurityGroup.NewAuthorizeSecurityGroupEgressParams()
					}

					p.SetSecuritygroupid(d.Id())
					p.SetUsersecuritygrouplist(map[string]string{sg.Account: usg.(string)})

					// Create a single rule
					err = createSecurityGroupRule(d, meta, rule, p, usg.(string))
					if err != nil {
						errs = multierror.Append(errs, err)
					}
				}
			}

			// If we have at least one UUID, we need to save the rule
			if len(rule["uuids"].(map[string]interface{})) > 0 {
				rules.Add(rule)
			}

			<-sem
		}(rule.(map[string]interface{}))
	}

	wg.Wait()

	return errs.ErrorOrNil()
}

func createSecurityGroupRule(d *schema.ResourceData, meta interface{}, rule map[string]interface{}, p authorizeSecurityGroupParams, uuid string) error {
	cs := meta.(*cloudstack.CloudStackClient)
	uuids := rule["uuids"].(map[string]interface{})

	// Set the protocol
	p.SetProtocol(rule["protocol"].(string))

	// If the protocol is ICMP set the needed ICMP parameters
	if rule["protocol"].(string) == "icmp" {
		p.SetIcmptype(rule["icmp_type"].(int))
		p.SetIcmpcode(rule["icmp_code"].(int))

		ruleID, err := createIngressOrEgressRule(cs, p)
		if err != nil {
			return err
		}

		uuids[uuid+"icmp"] = ruleID
		rule["uuids"] = uuids
	}

	// If protocol is TCP or UDP, loop through all ports
	if rule["protocol"].(string) == "tcp" || rule["protocol"].(string) == "udp" {
		if ps := rule["ports"].(*schema.Set); ps.Len() > 0 {

			// Create an empty schema.Set to hold all processed ports
			ports := &schema.Set{F: schema.HashString}

			for _, port := range ps.List() {
				if _, ok := uuids[uuid+port.(string)]; ok {
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

				ruleID, err := createIngressOrEgressRule(cs, p)
				if err != nil {
					return err
				}

				ports.Add(port)
				rule["ports"] = ports

				uuids[uuid+port.(string)] = ruleID
				rule["uuids"] = uuids
			}
		}
	}

	return nil
}

func createIngressOrEgressRule(cs *cloudstack.CloudStackClient, p authorizeSecurityGroupParams) (string, error) {
	switch p := p.(type) {
	case *cloudstack.AuthorizeSecurityGroupIngressParams:
		r, err := cs.SecurityGroup.AuthorizeSecurityGroupIngress(p)
		if err != nil {
			return "", err
		}
		return r.Ruleid, nil
	case *cloudstack.AuthorizeSecurityGroupEgressParams:
		r, err := cs.SecurityGroup.AuthorizeSecurityGroupEgress(p)
		if err != nil {
			return "", err
		}
		return r.Ruleid, nil
	default:
		return "", fmt.Errorf("Unknown authorize security group rule type: %v", p)
	}
}

func resourceCloudStackSecurityGroupRuleRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the security group details
	sg, count, err := cs.SecurityGroup.GetSecurityGroupByID(
		d.Id(),
		cloudstack.WithProject(d.Get("project").(string)),
	)
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] Security group %s does not longer exist", d.Get("name").(string))
			d.SetId("")
			return nil
		}

		return err
	}

	// Make a map of all the rule indexes so we can easily find a rule
	sgRules := append(sg.Ingressrule, sg.Egressrule...)
	ruleIndex := make(map[string]int, len(sgRules))
	for idx, r := range sgRules {
		ruleIndex[r.Ruleid] = idx
	}

	// Create an empty schema.Set to hold all rules
	rules := resourceCloudStackSecurityGroupRule().Schema["rule"].ZeroValue().(*schema.Set)

	// Read all rules that are configured
	if rs := d.Get("rule").(*schema.Set); rs.Len() > 0 {
		for _, rule := range rs.List() {
			rule := rule.(map[string]interface{})

			// First get any existing values
			cidrList, cidrListOK := rule["cidr_list"].(*schema.Set)
			usgList, usgListOk := rule["user_security_group_list"].(*schema.Set)

			// Then reset the values to a new empty set
			rule["cidr_list"] = &schema.Set{F: schema.HashString}
			rule["user_security_group_list"] = &schema.Set{F: schema.HashString}

			if cidrListOK && cidrList.Len() > 0 {
				for _, cidr := range cidrList.List() {
					readSecurityGroupRule(sg, ruleIndex, rule, cidr.(string))
				}
			}

			if usgListOk && usgList.Len() > 0 {
				for _, usg := range usgList.List() {
					readSecurityGroupRule(sg, ruleIndex, rule, usg.(string))
				}
			}

			rules.Add(rule)
		}
	}

	return nil
}

func readSecurityGroupRule(sg *cloudstack.SecurityGroup, ruleIndex map[string]int, rule map[string]interface{}, uuid string) {
	uuids := rule["uuids"].(map[string]interface{})
	sgRules := append(sg.Ingressrule, sg.Egressrule...)

	if rule["protocol"].(string) == "icmp" {
		id, ok := uuids[uuid+"icmp"]
		if !ok {
			return
		}

		// Get the rule
		idx, ok := ruleIndex[id.(string)]
		if !ok {
			delete(uuids, uuid+"icmp")
			return
		}

		r := sgRules[idx]

		// Update the values
		if r.Cidr != "" {
			rule["cidr_list"].(*schema.Set).Add(r.Cidr)
		}

		if r.Securitygroupname != "" {
			rule["user_security_group_list"].(*schema.Set).Add(r.Securitygroupname)
		}

		rule["protocol"] = r.Protocol
		rule["icmp_type"] = r.Icmptype
		rule["icmp_code"] = r.Icmpcode
	}

	// If protocol is tcp or udp, loop through all ports
	if rule["protocol"].(string) == "tcp" || rule["protocol"].(string) == "udp" {
		if ps := rule["ports"].(*schema.Set); ps.Len() > 0 {

			// Create an empty schema.Set to hold all ports
			ports := &schema.Set{F: schema.HashString}

			// Loop through all ports and retrieve their info
			for _, port := range ps.List() {
				id, ok := uuids[uuid+port.(string)]
				if !ok {
					continue
				}

				// Get the rule
				idx, ok := ruleIndex[id.(string)]
				if !ok {
					delete(uuids, uuid+port.(string))
					continue
				}

				r := sgRules[idx]

				// Create a set with all CIDR's
				cidrs := &schema.Set{F: schema.HashString}
				for _, cidr := range strings.Split(r.Cidr, ",") {
					cidrs.Add(cidr)
				}

				// Update the values
				rule["protocol"] = r.Protocol
				ports.Add(port)
			}

			// If there is at least one port found, add this rule to the rules set
			if ports.Len() > 0 {
				rule["ports"] = ports
			}
		}
	}
}

func resourceCloudStackSecurityGroupRuleUpdate(d *schema.ResourceData, meta interface{}) error {
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

		// First loop through all the old rules destroy them
		if ors.Len() > 0 {
			err := deleteSecurityGroupRules(d, meta, rules, ors)

			// We need to update this first to preserve the correct state
			d.Set("rule", rules)

			if err != nil {
				return err
			}
		}

		// Then loop through all the new rules and delete them
		if nrs.Len() > 0 {
			err := createSecurityGroupRules(d, meta, rules, nrs)

			// We need to update this first to preserve the correct state
			d.Set("rule", rules)

			if err != nil {
				return err
			}
		}
	}

	return resourceCloudStackSecurityGroupRuleRead(d, meta)
}

func resourceCloudStackSecurityGroupRuleDelete(d *schema.ResourceData, meta interface{}) error {
	// Create an empty rule set to hold all rules that where
	// not deleted correctly
	rules := resourceCloudStackSecurityGroupRule().Schema["rule"].ZeroValue().(*schema.Set)

	// Delete all rules
	if ors := d.Get("rule").(*schema.Set); ors.Len() > 0 {
		err := deleteSecurityGroupRules(d, meta, rules, ors)

		// We need to update this first to preserve the correct state
		d.Set("rule", rules)

		if err != nil {
			return err
		}
	}

	return nil
}

func deleteSecurityGroupRules(d *schema.ResourceData, meta interface{}, rules *schema.Set, ors *schema.Set) error {
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

			// Create a single rule
			err := deleteSecurityGroupRule(d, meta, rule)
			if err != nil {
				errs = multierror.Append(errs, err)
			}

			// If we have at least one UUID, we need to save the rule
			if len(rule["uuids"].(map[string]interface{})) > 0 {
				rules.Add(rule)
			}

			<-sem
		}(rule.(map[string]interface{}))
	}

	wg.Wait()

	return errs.ErrorOrNil()
}

func deleteSecurityGroupRule(d *schema.ResourceData, meta interface{}, rule map[string]interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	uuids := rule["uuids"].(map[string]interface{})

	for k, id := range uuids {
		// We don't care about the count here, so just continue
		if k == "%" {
			continue
		}

		var err error
		switch rule["traffic_type"].(string) {
		case "ingress":
			p := cs.SecurityGroup.NewRevokeSecurityGroupIngressParams(id.(string))
			_, err = cs.SecurityGroup.RevokeSecurityGroupIngress(p)
		case "egress":
			p := cs.SecurityGroup.NewRevokeSecurityGroupEgressParams(id.(string))
			_, err = cs.SecurityGroup.RevokeSecurityGroupEgress(p)
		}

		if err != nil {
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

	return nil
}

func verifySecurityGroupRuleParams(d *schema.ResourceData, rule map[string]interface{}) error {
	cidrList, cidrListOK := rule["cidr_list"].(*schema.Set)
	usgList, usgListOK := rule["user_security_group_list"].(*schema.Set)

	if (!cidrListOK || cidrList.Len() == 0) && (!usgListOK || usgList.Len() == 0) {
		return fmt.Errorf(
			"You must supply at least one 'cidr_list' or `user_security_group_ids` entry")
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
	case "tcp", "udp":
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
	default:
		_, err := strconv.ParseInt(protocol, 0, 0)
		if err != nil {
			return fmt.Errorf(
				"%q is not a valid protocol. Valid options are 'tcp', 'udp' and 'icmp'", protocol)
		}
	}

	traffic := rule["traffic_type"].(string)
	if traffic != "ingress" && traffic != "egress" {
		return fmt.Errorf(
			"Parameter traffic_type only accepts 'ingress' or 'egress' as values")
	}

	return nil
}
