package aws

import (
	"bytes"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsNetworkAclRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsNetworkAclRuleCreate,
		Read:   resourceAwsNetworkAclRuleRead,
		Delete: resourceAwsNetworkAclRuleDelete,

		Schema: map[string]*schema.Schema{
			"network_acl_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"rule_number": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"egress": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},
			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"rule_action": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"cidr_block": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"from_port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"to_port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"icmp_type": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"icmp_code": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
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
			return fmt.Errorf("Invalid Protocol %s for rule %#v", protocol, d.Get("rule_number").(int))
		}
	}
	log.Printf("[INFO] Transformed Protocol %s into %d", protocol, p)

	params := &ec2.CreateNetworkAclEntryInput{
		NetworkAclId: aws.String(d.Get("network_acl_id").(string)),
		Egress:       aws.Bool(d.Get("egress").(bool)),
		RuleNumber:   aws.Int64(int64(d.Get("rule_number").(int))),
		Protocol:     aws.String(strconv.Itoa(p)),
		CidrBlock:    aws.String(d.Get("cidr_block").(string)),
		RuleAction:   aws.String(d.Get("rule_action").(string)),
		PortRange: &ec2.PortRange{
			From: aws.Int64(int64(d.Get("from_port").(int))),
			To:   aws.Int64(int64(d.Get("to_port").(int))),
		},
	}

	// Specify additional required fields for ICMP
	if p == 1 {
		params.IcmpTypeCode = &ec2.IcmpTypeCode{}
		if v, ok := d.GetOk("icmp_code"); ok {
			params.IcmpTypeCode.Code = aws.Int64(int64(v.(int)))
		}
		if v, ok := d.GetOk("icmp_type"); ok {
			params.IcmpTypeCode.Type = aws.Int64(int64(v.(int)))
		}
	}

	log.Printf("[INFO] Creating Network Acl Rule: %d (%t)", d.Get("rule_number").(int), d.Get("egress").(bool))
	_, err := conn.CreateNetworkAclEntry(params)
	if err != nil {
		return fmt.Errorf("Error Creating Network Acl Rule: %s", err.Error())
	}
	d.SetId(networkAclIdRuleNumberEgressHash(d.Get("network_acl_id").(string), d.Get("rule_number").(int), d.Get("egress").(bool), d.Get("protocol").(string)))
	return resourceAwsNetworkAclRuleRead(d, meta)
}

func resourceAwsNetworkAclRuleRead(d *schema.ResourceData, meta interface{}) error {
	resp, err := findNetworkAclRule(d, meta)
	if err != nil {
		return err
	}

	d.Set("rule_number", resp.RuleNumber)
	d.Set("cidr_block", resp.CidrBlock)
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
			return fmt.Errorf("Invalid Protocol %s for rule %#v", *resp.Protocol, d.Get("rule_number").(int))
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
		Values: []*string{aws.String(fmt.Sprintf("%v", d.Get("rule_number").(int)))},
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

	if resp == nil || len(resp.NetworkAcls) != 1 || resp.NetworkAcls[0] == nil {
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
