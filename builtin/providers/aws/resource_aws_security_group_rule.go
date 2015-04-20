package aws

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSecurityGroupRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSecurityGroupRuleCreate,
		Read:   resourceAwsSecurityGroupRuleRead,
		Delete: resourceAwsSecurityGroupRuleDelete,

		Schema: map[string]*schema.Schema{
			"type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Type of rule, ingress (inbound) or egress (outbound).",
			},

			"from_port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"to_port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cidr_blocks": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"security_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"source_security_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"self": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsSecurityGroupRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	sg_id := d.Get("security_group_id").(string)
	sg, err := findResourceSecurityGroup(conn, sg_id)

	if err != nil {
		return fmt.Errorf("sorry")
	}

	perm := expandIPPerm(d, sg)

	ruleType := d.Get("type").(string)

	switch ruleType {
	case "ingress":
		log.Printf("[DEBUG] Authorizing security group %s %s rule: %#v",
			sg_id, "Ingress", perm)

		req := &ec2.AuthorizeSecurityGroupIngressInput{
			GroupID:       sg.GroupID,
			IPPermissions: []*ec2.IPPermission{perm},
		}

		if sg.VPCID == nil || *sg.VPCID == "" {
			req.GroupID = nil
			req.GroupName = sg.GroupName
		}

		_, err := conn.AuthorizeSecurityGroupIngress(req)

		if err != nil {
			return fmt.Errorf(
				"Error authorizing security group %s rules: %s",
				"rules", err)
		}

	case "egress":
		log.Printf("[DEBUG] Authorizing security group %s %s rule: %#v",
			sg_id, "Egress", perm)

		req := &ec2.AuthorizeSecurityGroupEgressInput{
			GroupID:       sg.GroupID,
			IPPermissions: []*ec2.IPPermission{perm},
		}

		_, err = conn.AuthorizeSecurityGroupEgress(req)

		if err != nil {
			return fmt.Errorf(
				"Error authorizing security group %s rules: %s",
				"rules", err)
		}

	default:
		return fmt.Errorf("Security Group Rule must be type 'ingress' or type 'egress'")
	}

	d.SetId(ipPermissionIDHash(ruleType, perm))

	return resourceAwsSecurityGroupRuleRead(d, meta)
}

func resourceAwsSecurityGroupRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	sg_id := d.Get("security_group_id").(string)
	sg, err := findResourceSecurityGroup(conn, sg_id)
	if err != nil {
		d.SetId("")
	}

	var rule *ec2.IPPermission
	ruleType := d.Get("type").(string)
	var rl []*ec2.IPPermission
	switch ruleType {
	case "ingress":
		rl = sg.IPPermissions
	default:
		rl = sg.IPPermissionsEgress
	}

	for _, r := range rl {
		if d.Id() == ipPermissionIDHash(ruleType, r) {
			rule = r
		}
	}

	if rule == nil {
		log.Printf("[DEBUG] Unable to find matching %s Security Group Rule for Group %s",
			ruleType, sg_id)
		d.SetId("")
		return nil
	}

	d.Set("from_port", rule.FromPort)
	d.Set("to_port", rule.ToPort)
	d.Set("protocol", rule.IPProtocol)
	d.Set("type", ruleType)

	var cb []string
	for _, c := range rule.IPRanges {
		cb = append(cb, *c.CIDRIP)
	}

	d.Set("cidr_blocks", cb)

	if len(rule.UserIDGroupPairs) > 0 {
		s := rule.UserIDGroupPairs[0]
		d.Set("source_security_group_id", *s.GroupID)
	}

	return nil
}

func resourceAwsSecurityGroupRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	sg_id := d.Get("security_group_id").(string)
	sg, err := findResourceSecurityGroup(conn, sg_id)

	if err != nil {
		return fmt.Errorf("sorry")
	}

	perm := expandIPPerm(d, sg)
	ruleType := d.Get("type").(string)
	switch ruleType {
	case "ingress":
		log.Printf("[DEBUG] Revoking security group %#v %s rule: %#v",
			sg_id, "ingress", perm)
		req := &ec2.RevokeSecurityGroupIngressInput{
			GroupID:       sg.GroupID,
			IPPermissions: []*ec2.IPPermission{perm},
		}

		_, err = conn.RevokeSecurityGroupIngress(req)

		if err != nil {
			return fmt.Errorf(
				"Error revoking security group %s rules: %s",
				sg_id, err)
		}
	case "egress":

		log.Printf("[DEBUG] Revoking security group %#v %s rule: %#v",
			sg_id, "egress", perm)
		req := &ec2.RevokeSecurityGroupEgressInput{
			GroupID:       sg.GroupID,
			IPPermissions: []*ec2.IPPermission{perm},
		}

		_, err = conn.RevokeSecurityGroupEgress(req)

		if err != nil {
			return fmt.Errorf(
				"Error revoking security group %s rules: %s",
				sg_id, err)
		}
	}

	d.SetId("")

	return nil
}

func findResourceSecurityGroup(conn *ec2.EC2, id string) (*ec2.SecurityGroup, error) {
	req := &ec2.DescribeSecurityGroupsInput{
		GroupIDs: []*string{aws.String(id)},
	}
	resp, err := conn.DescribeSecurityGroups(req)
	if err != nil {
		if ec2err, ok := err.(aws.APIError); ok {
			if ec2err.Code == "InvalidSecurityGroupID.NotFound" ||
				ec2err.Code == "InvalidGroup.NotFound" {
				resp = nil
				err = nil
			}
		}

		if err != nil {
			log.Printf("Error on findResourceSecurityGroup: %s", err)
			return nil, err
		}
	}

	return resp.SecurityGroups[0], nil
}

func ipPermissionIDHash(ruleType string, ip *ec2.IPPermission) string {
	var buf bytes.Buffer
	// for egress rules, an TCP rule of -1 is automatically added, in which case
	// the to and from ports will be nil. We don't record this rule locally.
	if ip.IPProtocol != nil && *ip.IPProtocol != "-1" {
		buf.WriteString(fmt.Sprintf("%d-", *ip.FromPort))
		buf.WriteString(fmt.Sprintf("%d-", *ip.ToPort))
		buf.WriteString(fmt.Sprintf("%s-", *ip.IPProtocol))
	}
	buf.WriteString(fmt.Sprintf("%s-", ruleType))

	// We need to make sure to sort the strings below so that we always
	// generate the same hash code no matter what is in the set.
	if len(ip.IPRanges) > 0 {
		s := make([]string, len(ip.IPRanges))
		for i, r := range ip.IPRanges {
			s[i] = *r.CIDRIP
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}

	return fmt.Sprintf("sg-%d", hashcode.String(buf.String()))
}

func expandIPPerm(d *schema.ResourceData, sg *ec2.SecurityGroup) *ec2.IPPermission {
	var perm ec2.IPPermission

	perm.FromPort = aws.Long(int64(d.Get("from_port").(int)))
	perm.ToPort = aws.Long(int64(d.Get("to_port").(int)))
	perm.IPProtocol = aws.String(d.Get("protocol").(string))

	var groups []string
	if raw, ok := d.GetOk("source_security_group_id"); ok {
		groups = append(groups, raw.(string))
	}

	if v, ok := d.GetOk("self"); ok && v.(bool) {
		if sg.VPCID != nil && *sg.VPCID != "" {
			groups = append(groups, *sg.GroupID)
		} else {
			groups = append(groups, *sg.GroupName)
		}
	}

	if len(groups) > 0 {
		perm.UserIDGroupPairs = make([]*ec2.UserIDGroupPair, len(groups))
		for i, name := range groups {
			ownerId, id := "", name
			if items := strings.Split(id, "/"); len(items) > 1 {
				ownerId, id = items[0], items[1]
			}

			perm.UserIDGroupPairs[i] = &ec2.UserIDGroupPair{
				GroupID: aws.String(id),
				UserID:  aws.String(ownerId),
			}

			if sg.VPCID == nil || *sg.VPCID == "" {
				perm.UserIDGroupPairs[i].GroupID = nil
				perm.UserIDGroupPairs[i].GroupName = aws.String(id)
				perm.UserIDGroupPairs[i].UserID = nil
			}
		}
	}

	if raw, ok := d.GetOk("cidr_blocks"); ok {
		list := raw.([]interface{})
		perm.IPRanges = make([]*ec2.IPRange, len(list))
		for i, v := range list {
			perm.IPRanges[i] = &ec2.IPRange{CIDRIP: aws.String(v.(string))}
		}
	}

	return &perm
}
