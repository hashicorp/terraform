package aws

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsNetworkAclRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsNetworkAclRuleCreate,
		Read:   resourceAwsNetworkAclRuleRead,
		Delete: resourceAwsNetworkAclRuleDelete,

		Schema: map[string]*schema.Schema{
			"network_acl_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"rule_number": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"egress": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},
			"protocol": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					pi := protocolIntegers()
					if val, ok := pi[old]; ok {
						old = strconv.Itoa(val)
					}
					if val, ok := pi[new]; ok {
						new = strconv.Itoa(val)
					}

					return old == new
				},
			},
			"rule_action": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"cidr_block": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"ipv6_cidr_block"},
			},
			"ipv6_cidr_block": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"cidr_block"},
			},
			"from_port": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"to_port": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"icmp_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateICMPArgumentValue,
			},
			"icmp_code": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateICMPArgumentValue,
			},
		},
	}
}

func resourceAwsNetworkAclRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	protocol := d.Get("protocol").(string)
	p, protocolErr := strconv.Atoi(protocol)
	if protocolErr != nil {
		var ok bool
		p, ok = protocolIntegers()[protocol]
		if !ok {
			return fmt.Errorf("Invalid Protocol %s for rule %d", protocol, d.Get("rule_number").(int))
		}
	}
	log.Printf("[INFO] Transformed Protocol %s into %d", protocol, p)

	params := &ec2.CreateNetworkAclEntryInput{
		NetworkAclId: aws.String(d.Get("network_acl_id").(string)),
		Egress:       aws.Bool(d.Get("egress").(bool)),
		RuleNumber:   aws.Int64(int64(d.Get("rule_number").(int))),
		Protocol:     aws.String(strconv.Itoa(p)),
		RuleAction:   aws.String(d.Get("rule_action").(string)),
		PortRange: &ec2.PortRange{
			From: aws.Int64(int64(d.Get("from_port").(int))),
			To:   aws.Int64(int64(d.Get("to_port").(int))),
		},
	}

	cidr, hasCidr := d.GetOk("cidr_block")
	ipv6Cidr, hasIpv6Cidr := d.GetOk("ipv6_cidr_block")

	if hasCidr == false && hasIpv6Cidr == false {
		return fmt.Errorf("Either `cidr_block` or `ipv6_cidr_block` must be defined")
	}

	if hasCidr {
		params.CidrBlock = aws.String(cidr.(string))
	}

	if hasIpv6Cidr {
		params.Ipv6CidrBlock = aws.String(ipv6Cidr.(string))
	}

	// Specify additional required fields for ICMP. For the list
	// of ICMP codes and types, see: https://www.iana.org/assignments/icmp-parameters/icmp-parameters.xhtml
	if p == 1 || p == 58 {
		params.IcmpTypeCode = &ec2.IcmpTypeCode{}
		if v, ok := d.GetOk("icmp_type"); ok {
			icmpType, err := strconv.Atoi(v.(string))
			if err != nil {
				return fmt.Errorf("Unable to parse ICMP type %s for rule %d", v, d.Get("rule_number").(int))
			}
			params.IcmpTypeCode.Type = aws.Int64(int64(icmpType))
			log.Printf("[DEBUG] Got ICMP type %d for rule %d", icmpType, d.Get("rule_number").(int))
		}
		if v, ok := d.GetOk("icmp_code"); ok {
			icmpCode, err := strconv.Atoi(v.(string))
			if err != nil {
				return fmt.Errorf("Unable to parse ICMP code %s for rule %d", v, d.Get("rule_number").(int))
			}
			params.IcmpTypeCode.Code = aws.Int64(int64(icmpCode))
			log.Printf("[DEBUG] Got ICMP code %d for rule %d", icmpCode, d.Get("rule_number").(int))
		}
	}

	log.Printf("[INFO] Creating Network Acl Rule: %d (%t)", d.Get("rule_number").(int), d.Get("egress").(bool))
	_, err := conn.CreateNetworkAclEntry(params)
	if err != nil {
		return fmt.Errorf("Error Creating Network Acl Rule: %s", err.Error())
	}
	d.SetId(networkAclIdRuleNumberEgressHash(d.Get("network_acl_id").(string), d.Get("rule_number").(int), d.Get("egress").(bool), d.Get("protocol").(string)))

	// It appears it might be a while until the newly created rule is visible via the
	// API (see issue GH-4721). Retry the `findNetworkAclRule` function until it is
	// visible (which in most cases is likely immediately).
	err = resource.Retry(3*time.Minute, func() *resource.RetryError {
		r, findErr := findNetworkAclRule(d, meta)
		if findErr != nil {
			return resource.RetryableError(findErr)
		}
		if r == nil {
			err := fmt.Errorf("Network ACL rule (%s) not found", d.Id())
			return resource.RetryableError(err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("Created Network ACL Rule was not visible in API within 3 minute period. Running 'terraform apply' again will resume infrastructure creation.")
	}

	return resourceAwsNetworkAclRuleRead(d, meta)
}

func resourceAwsNetworkAclRuleRead(d *schema.ResourceData, meta interface{}) error {
	resp, err := findNetworkAclRule(d, meta)
	if err != nil {
		return err
	}
	if resp == nil {
		log.Printf("[DEBUG] Network ACL rule (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("rule_number", resp.RuleNumber)
	d.Set("cidr_block", resp.CidrBlock)
	d.Set("ipv6_cidr_block", resp.Ipv6CidrBlock)
	d.Set("egress", resp.Egress)
	if resp.IcmpTypeCode != nil {
		d.Set("icmp_code", resp.IcmpTypeCode.Code)
		d.Set("icmp_type", resp.IcmpTypeCode.Type)
	}
	if resp.PortRange != nil {
		d.Set("from_port", resp.PortRange.From)
		d.Set("to_port", resp.PortRange.To)
	}

	d.Set("rule_action", resp.RuleAction)

	p, protocolErr := strconv.Atoi(*resp.Protocol)
	log.Printf("[INFO] Converting the protocol %v", p)
	if protocolErr == nil {
		var ok bool
		protocol, ok := protocolStrings(protocolIntegers())[p]
		if !ok {
			return fmt.Errorf("Invalid Protocol %s for rule %d", *resp.Protocol, d.Get("rule_number").(int))
		}
		log.Printf("[INFO] Transformed Protocol %s back into %s", *resp.Protocol, protocol)
		d.Set("protocol", protocol)
	}

	return nil
}

func resourceAwsNetworkAclRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	params := &ec2.DeleteNetworkAclEntryInput{
		NetworkAclId: aws.String(d.Get("network_acl_id").(string)),
		RuleNumber:   aws.Int64(int64(d.Get("rule_number").(int))),
		Egress:       aws.Bool(d.Get("egress").(bool)),
	}

	log.Printf("[INFO] Deleting Network Acl Rule: %s", d.Id())
	_, err := conn.DeleteNetworkAclEntry(params)
	if err != nil {
		return fmt.Errorf("Error Deleting Network Acl Rule: %s", err.Error())
	}

	return nil
}

func findNetworkAclRule(d *schema.ResourceData, meta interface{}) (*ec2.NetworkAclEntry, error) {
	conn := meta.(*AWSClient).ec2conn

	filters := make([]*ec2.Filter, 0, 2)
	ruleNumberFilter := &ec2.Filter{
		Name:   aws.String("entry.rule-number"),
		Values: []*string{aws.String(fmt.Sprintf("%d", d.Get("rule_number").(int)))},
	}
	filters = append(filters, ruleNumberFilter)
	egressFilter := &ec2.Filter{
		Name:   aws.String("entry.egress"),
		Values: []*string{aws.String(fmt.Sprintf("%v", d.Get("egress").(bool)))},
	}
	filters = append(filters, egressFilter)
	params := &ec2.DescribeNetworkAclsInput{
		NetworkAclIds: []*string{aws.String(d.Get("network_acl_id").(string))},
		Filters:       filters,
	}

	log.Printf("[INFO] Describing Network Acl: %s", d.Get("network_acl_id").(string))
	log.Printf("[INFO] Describing Network Acl with the Filters %#v", params)
	resp, err := conn.DescribeNetworkAcls(params)
	if err != nil {
		return nil, fmt.Errorf("Error Finding Network Acl Rule %d: %s", d.Get("rule_number").(int), err.Error())
	}

	if resp == nil || len(resp.NetworkAcls) == 0 || resp.NetworkAcls[0] == nil {
		// Missing NACL rule.
		return nil, nil
	}
	if len(resp.NetworkAcls) > 1 {
		return nil, fmt.Errorf(
			"Expected to find one Network ACL, got: %#v",
			resp.NetworkAcls)
	}
	networkAcl := resp.NetworkAcls[0]
	if networkAcl.Entries != nil {
		for _, i := range networkAcl.Entries {
			if *i.RuleNumber == int64(d.Get("rule_number").(int)) && *i.Egress == d.Get("egress").(bool) {
				return i, nil
			}
		}
	}
	return nil, fmt.Errorf(
		"Expected the Network ACL to have Entries, got: %#v",
		networkAcl)

}

func networkAclIdRuleNumberEgressHash(networkAclId string, ruleNumber int, egress bool, protocol string) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%s-", networkAclId))
	buf.WriteString(fmt.Sprintf("%d-", ruleNumber))
	buf.WriteString(fmt.Sprintf("%t-", egress))
	buf.WriteString(fmt.Sprintf("%s-", protocol))
	return fmt.Sprintf("nacl-%d", hashcode.String(buf.String()))
}

func validateICMPArgumentValue(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	_, err := strconv.Atoi(value)
	if len(value) == 0 || err != nil {
		errors = append(errors, fmt.Errorf("%q must be an integer value: %q", k, value))
	}
	return
}
