package aws

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSecurityGroupRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSecurityGroupRuleCreate,
		Read:   resourceAwsSecurityGroupRuleRead,
		Delete: resourceAwsSecurityGroupRuleDelete,

		SchemaVersion: 2,
		MigrateState:  resourceAwsSecurityGroupRuleMigrateState,

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
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: protocolStateFunc,
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
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Computed:      true,
				ConflictsWith: []string{"cidr_blocks", "self"},
			},

			"self": &schema.Schema{
				Type:          schema.TypeBool,
				Optional:      true,
				Default:       false,
				ForceNew:      true,
				ConflictsWith: []string{"cidr_blocks"},
			},
		},
	}
}

func resourceAwsSecurityGroupRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	sg_id := d.Get("security_group_id").(string)

	awsMutexKV.Lock(sg_id)
	defer awsMutexKV.Unlock(sg_id)

	sg, err := findResourceSecurityGroup(conn, sg_id)
	if err != nil {
		return err
	}

	perm, err := expandIPPerm(d, sg)
	if err != nil {
		return err
	}

	ruleType := d.Get("type").(string)
	isVPC := sg.VpcId != nil && *sg.VpcId != ""

	var autherr error
	switch ruleType {
	case "ingress":
		log.Printf("[DEBUG] Authorizing security group %s %s rule: %s",
			sg_id, "Ingress", perm)

		req := &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId:       sg.GroupId,
			IpPermissions: []*ec2.IpPermission{perm},
		}

		if !isVPC {
			req.GroupId = nil
			req.GroupName = sg.GroupName
		}

		_, autherr = conn.AuthorizeSecurityGroupIngress(req)

	case "egress":
		log.Printf("[DEBUG] Authorizing security group %s %s rule: %#v",
			sg_id, "Egress", perm)

		req := &ec2.AuthorizeSecurityGroupEgressInput{
			GroupId:       sg.GroupId,
			IpPermissions: []*ec2.IpPermission{perm},
		}

		_, autherr = conn.AuthorizeSecurityGroupEgress(req)

	default:
		return fmt.Errorf("Security Group Rule must be type 'ingress' or type 'egress'")
	}

	if autherr != nil {
		if awsErr, ok := autherr.(awserr.Error); ok {
			if awsErr.Code() == "InvalidPermission.Duplicate" {
				return fmt.Errorf(`[WARN] A duplicate Security Group rule was found on (%s). This may be
a side effect of a now-fixed Terraform issue causing two security groups with
identical attributes but different source_security_group_ids to overwrite each
other in the state. See https://github.com/hashicorp/terraform/pull/2376 for more
information and instructions for recovery. Error message: %s`, sg_id, awsErr.Message())
			}
		}

		return fmt.Errorf(
			"Error authorizing security group rule type %s: %s",
			ruleType, autherr)
	}

	id := ipPermissionIDHash(sg_id, ruleType, perm)
	log.Printf("[DEBUG] Computed group rule ID %s", id)

	retErr := resource.Retry(5*time.Minute, func() *resource.RetryError {
		sg, err := findResourceSecurityGroup(conn, sg_id)

		if err != nil {
			log.Printf("[DEBUG] Error finding Secuirty Group (%s) for Rule (%s): %s", sg_id, id, err)
			return resource.NonRetryableError(err)
		}

		var rules []*ec2.IpPermission
		switch ruleType {
		case "ingress":
			rules = sg.IpPermissions
		default:
			rules = sg.IpPermissionsEgress
		}

		rule := findRuleMatch(perm, rules, isVPC)

		if rule == nil {
			log.Printf("[DEBUG] Unable to find matching %s Security Group Rule (%s) for Group %s",
				ruleType, id, sg_id)
			return resource.RetryableError(fmt.Errorf("No match found"))
		}

		log.Printf("[DEBUG] Found rule for Security Group Rule (%s): %s", id, rule)
		return nil
	})

	if retErr != nil {
		log.Printf("[DEBUG] Error finding matching %s Security Group Rule (%s) for Group %s -- NO STATE WILL BE SAVED",
			ruleType, id, sg_id)
		return nil
	}

	d.SetId(id)
	return nil
}

func resourceAwsSecurityGroupRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	sg_id := d.Get("security_group_id").(string)
	sg, err := findResourceSecurityGroup(conn, sg_id)
	if err != nil {
		log.Printf("[DEBUG] Error finding Secuirty Group (%s) for Rule (%s): %s", sg_id, d.Id(), err)
		d.SetId("")
		return nil
	}

	isVPC := sg.VpcId != nil && *sg.VpcId != ""

	var rule *ec2.IpPermission
	var rules []*ec2.IpPermission
	ruleType := d.Get("type").(string)
	switch ruleType {
	case "ingress":
		rules = sg.IpPermissions
	default:
		rules = sg.IpPermissionsEgress
	}

	p, err := expandIPPerm(d, sg)
	if err != nil {
		return err
	}

	if len(rules) == 0 {
		log.Printf("[WARN] No %s rules were found for Security Group (%s) looking for Security Group Rule (%s)",
			ruleType, *sg.GroupName, d.Id())
		d.SetId("")
		return nil
	}

	rule = findRuleMatch(p, rules, isVPC)

	if rule == nil {
		log.Printf("[DEBUG] Unable to find matching %s Security Group Rule (%s) for Group %s",
			ruleType, d.Id(), sg_id)
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Found rule for Security Group Rule (%s): %s", d.Id(), rule)

	d.Set("from_port", rule.FromPort)
	d.Set("to_port", rule.ToPort)
	d.Set("protocol", rule.IpProtocol)
	d.Set("type", ruleType)

	var cb []string
	for _, c := range p.IpRanges {
		cb = append(cb, *c.CidrIp)
	}

	d.Set("cidr_blocks", cb)

	if len(p.UserIdGroupPairs) > 0 {
		s := p.UserIdGroupPairs[0]
		if isVPC {
			d.Set("source_security_group_id", *s.GroupId)
		} else {
			d.Set("source_security_group_id", *s.GroupName)
		}
	}

	return nil
}

func resourceAwsSecurityGroupRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	sg_id := d.Get("security_group_id").(string)

	awsMutexKV.Lock(sg_id)
	defer awsMutexKV.Unlock(sg_id)

	sg, err := findResourceSecurityGroup(conn, sg_id)
	if err != nil {
		return err
	}

	perm, err := expandIPPerm(d, sg)
	if err != nil {
		return err
	}
	ruleType := d.Get("type").(string)
	switch ruleType {
	case "ingress":
		log.Printf("[DEBUG] Revoking rule (%s) from security group %s:\n%s",
			"ingress", sg_id, perm)
		req := &ec2.RevokeSecurityGroupIngressInput{
			GroupId:       sg.GroupId,
			IpPermissions: []*ec2.IpPermission{perm},
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
			GroupId:       sg.GroupId,
			IpPermissions: []*ec2.IpPermission{perm},
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
		GroupIds: []*string{aws.String(id)},
	}
	resp, err := conn.DescribeSecurityGroups(req)
	if err != nil {
		return nil, err
	}

	if resp == nil || len(resp.SecurityGroups) != 1 || resp.SecurityGroups[0] == nil {
		return nil, fmt.Errorf(
			"Expected to find one security group with ID %q, got: %#v",
			id, resp.SecurityGroups)
	}

	return resp.SecurityGroups[0], nil
}

// ByGroupPair implements sort.Interface for []*ec2.UserIDGroupPairs based on
// GroupID or GroupName field (only one should be set).
type ByGroupPair []*ec2.UserIdGroupPair

func (b ByGroupPair) Len() int      { return len(b) }
func (b ByGroupPair) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByGroupPair) Less(i, j int) bool {
	if b[i].GroupId != nil && b[j].GroupId != nil {
		return *b[i].GroupId < *b[j].GroupId
	}
	if b[i].GroupName != nil && b[j].GroupName != nil {
		return *b[i].GroupName < *b[j].GroupName
	}

	panic("mismatched security group rules, may be a terraform bug")
}

func findRuleMatch(p *ec2.IpPermission, rules []*ec2.IpPermission, isVPC bool) *ec2.IpPermission {
	var rule *ec2.IpPermission
	for _, r := range rules {
		if r.ToPort != nil && *p.ToPort != *r.ToPort {
			continue
		}

		if r.FromPort != nil && *p.FromPort != *r.FromPort {
			continue
		}

		if r.IpProtocol != nil && *p.IpProtocol != *r.IpProtocol {
			continue
		}

		remaining := len(p.IpRanges)
		for _, ip := range p.IpRanges {
			for _, rip := range r.IpRanges {
				if *ip.CidrIp == *rip.CidrIp {
					remaining--
				}
			}
		}

		if remaining > 0 {
			continue
		}

		remaining = len(p.UserIdGroupPairs)
		for _, ip := range p.UserIdGroupPairs {
			for _, rip := range r.UserIdGroupPairs {
				if isVPC {
					if *ip.GroupId == *rip.GroupId {
						remaining--
					}
				} else {
					if *ip.GroupName == *rip.GroupName {
						remaining--
					}
				}
			}
		}

		if remaining > 0 {
			continue
		}

		rule = r
	}
	return rule
}

func ipPermissionIDHash(sg_id, ruleType string, ip *ec2.IpPermission) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%s-", sg_id))
	if ip.FromPort != nil && *ip.FromPort > 0 {
		buf.WriteString(fmt.Sprintf("%d-", *ip.FromPort))
	}
	if ip.ToPort != nil && *ip.ToPort > 0 {
		buf.WriteString(fmt.Sprintf("%d-", *ip.ToPort))
	}
	buf.WriteString(fmt.Sprintf("%s-", *ip.IpProtocol))
	buf.WriteString(fmt.Sprintf("%s-", ruleType))

	// We need to make sure to sort the strings below so that we always
	// generate the same hash code no matter what is in the set.
	if len(ip.IpRanges) > 0 {
		s := make([]string, len(ip.IpRanges))
		for i, r := range ip.IpRanges {
			s[i] = *r.CidrIp
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}

	if len(ip.UserIdGroupPairs) > 0 {
		sort.Sort(ByGroupPair(ip.UserIdGroupPairs))
		for _, pair := range ip.UserIdGroupPairs {
			if pair.GroupId != nil {
				buf.WriteString(fmt.Sprintf("%s-", *pair.GroupId))
			} else {
				buf.WriteString("-")
			}
			if pair.GroupName != nil {
				buf.WriteString(fmt.Sprintf("%s-", *pair.GroupName))
			} else {
				buf.WriteString("-")
			}
		}
	}

	return fmt.Sprintf("sgrule-%d", hashcode.String(buf.String()))
}

func expandIPPerm(d *schema.ResourceData, sg *ec2.SecurityGroup) (*ec2.IpPermission, error) {
	var perm ec2.IpPermission

	perm.FromPort = aws.Int64(int64(d.Get("from_port").(int)))
	perm.ToPort = aws.Int64(int64(d.Get("to_port").(int)))
	protocol := protocolForValue(d.Get("protocol").(string))
	perm.IpProtocol = aws.String(protocol)

	// build a group map that behaves like a set
	groups := make(map[string]bool)
	if raw, ok := d.GetOk("source_security_group_id"); ok {
		groups[raw.(string)] = true
	}

	if v, ok := d.GetOk("self"); ok && v.(bool) {
		// if sg.GroupId != nil {
		if sg.VpcId != nil && *sg.VpcId != "" {
			groups[*sg.GroupId] = true
		} else {
			groups[*sg.GroupName] = true
		}
	}

	if len(groups) > 0 {
		perm.UserIdGroupPairs = make([]*ec2.UserIdGroupPair, len(groups))
		// build string list of group name/ids
		var gl []string
		for k, _ := range groups {
			gl = append(gl, k)
		}

		for i, name := range gl {
			ownerId, id := "", name
			if items := strings.Split(id, "/"); len(items) > 1 {
				ownerId, id = items[0], items[1]
			}

			perm.UserIdGroupPairs[i] = &ec2.UserIdGroupPair{
				GroupId: aws.String(id),
				UserId:  aws.String(ownerId),
			}

			if sg.VpcId == nil || *sg.VpcId == "" {
				perm.UserIdGroupPairs[i].GroupId = nil
				perm.UserIdGroupPairs[i].GroupName = aws.String(id)
				perm.UserIdGroupPairs[i].UserId = nil
			}
		}
	}

	if raw, ok := d.GetOk("cidr_blocks"); ok {
		list := raw.([]interface{})
		perm.IpRanges = make([]*ec2.IpRange, len(list))
		for i, v := range list {
			cidrIP, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("empty element found in cidr_blocks - consider using the compact function")
			}
			perm.IpRanges[i] = &ec2.IpRange{CidrIp: aws.String(cidrIP)}
		}
	}

	return &perm, nil
}
