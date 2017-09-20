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
	"github.com/hashicorp/errwrap"
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
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Type of rule, ingress (inbound) or egress (outbound).",
				ValidateFunc: validateSecurityRuleType,
			},

			"from_port": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"to_port": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"protocol": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: protocolStateFunc,
			},

			"cidr_blocks": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateCIDRNetworkAddress,
				},
			},

			"ipv6_cidr_blocks": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateCIDRNetworkAddress,
				},
			},

			"prefix_list_ids": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"security_group_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"source_security_group_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Computed:      true,
				ConflictsWith: []string{"cidr_blocks", "self"},
			},

			"self": {
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

	// Verify that either 'cidr_blocks', 'self', or 'source_security_group_id' is set
	// If they are not set the AWS API will silently fail. This causes TF to hit a timeout
	// at 5-minutes waiting for the security group rule to appear, when it was never actually
	// created.
	if err := validateAwsSecurityGroupRule(d); err != nil {
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
			log.Printf("[DEBUG] Error finding Security Group (%s) for Rule (%s): %s", sg_id, id, err)
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
		return fmt.Errorf("Error finding matching %s Security Group Rule (%s) for Group %s",
			ruleType, id, sg_id)
	}

	d.SetId(id)
	return nil
}

func resourceAwsSecurityGroupRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	sg_id := d.Get("security_group_id").(string)
	sg, err := findResourceSecurityGroup(conn, sg_id)
	if _, notFound := err.(securityGroupNotFound); notFound {
		// The security group containing this rule no longer exists.
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error finding security group (%s) for rule (%s): %s", sg_id, d.Id(), err)
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

	d.Set("type", ruleType)
	if err := setFromIPPerm(d, sg, p); err != nil {
		return errwrap.Wrapf("Error setting IP Permission for Security Group Rule: {{err}}", err)
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
	if err, ok := err.(awserr.Error); ok && err.Code() == "InvalidGroup.NotFound" {
		return nil, securityGroupNotFound{id, nil}
	}
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, securityGroupNotFound{id, nil}
	}
	if len(resp.SecurityGroups) != 1 || resp.SecurityGroups[0] == nil {
		return nil, securityGroupNotFound{id, resp.SecurityGroups}
	}

	return resp.SecurityGroups[0], nil
}

type securityGroupNotFound struct {
	id             string
	securityGroups []*ec2.SecurityGroup
}

func (err securityGroupNotFound) Error() string {
	if err.securityGroups == nil {
		return fmt.Sprintf("No security group with ID %q", err.id)
	}
	return fmt.Sprintf("Expected to find one security group with ID %q, got: %#v",
		err.id, err.securityGroups)
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

		remaining = len(p.Ipv6Ranges)
		for _, ipv6 := range p.Ipv6Ranges {
			for _, ipv6ip := range r.Ipv6Ranges {
				if *ipv6.CidrIpv6 == *ipv6ip.CidrIpv6 {
					remaining--
				}
			}
		}

		if remaining > 0 {
			continue
		}

		remaining = len(p.PrefixListIds)
		for _, pl := range p.PrefixListIds {
			for _, rpl := range r.PrefixListIds {
				if *pl.PrefixListId == *rpl.PrefixListId {
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

	if len(ip.Ipv6Ranges) > 0 {
		s := make([]string, len(ip.Ipv6Ranges))
		for i, r := range ip.Ipv6Ranges {
			s[i] = *r.CidrIpv6
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}

	if len(ip.PrefixListIds) > 0 {
		s := make([]string, len(ip.PrefixListIds))
		for i, pl := range ip.PrefixListIds {
			s[i] = *pl.PrefixListId
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

	if raw, ok := d.GetOk("ipv6_cidr_blocks"); ok {
		list := raw.([]interface{})
		perm.Ipv6Ranges = make([]*ec2.Ipv6Range, len(list))
		for i, v := range list {
			cidrIP, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("empty element found in ipv6_cidr_blocks - consider using the compact function")
			}
			perm.Ipv6Ranges[i] = &ec2.Ipv6Range{CidrIpv6: aws.String(cidrIP)}
		}
	}

	if raw, ok := d.GetOk("prefix_list_ids"); ok {
		list := raw.([]interface{})
		perm.PrefixListIds = make([]*ec2.PrefixListId, len(list))
		for i, v := range list {
			prefixListID, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("empty element found in prefix_list_ids - consider using the compact function")
			}
			perm.PrefixListIds[i] = &ec2.PrefixListId{PrefixListId: aws.String(prefixListID)}
		}
	}

	return &perm, nil
}

func setFromIPPerm(d *schema.ResourceData, sg *ec2.SecurityGroup, rule *ec2.IpPermission) error {
	isVPC := sg.VpcId != nil && *sg.VpcId != ""

	d.Set("from_port", rule.FromPort)
	d.Set("to_port", rule.ToPort)
	d.Set("protocol", rule.IpProtocol)

	var cb []string
	for _, c := range rule.IpRanges {
		cb = append(cb, *c.CidrIp)
	}

	d.Set("cidr_blocks", cb)

	var ipv6 []string
	for _, ip := range rule.Ipv6Ranges {
		ipv6 = append(ipv6, *ip.CidrIpv6)
	}
	d.Set("ipv6_cidr_blocks", ipv6)

	var pl []string
	for _, p := range rule.PrefixListIds {
		pl = append(pl, *p.PrefixListId)
	}
	d.Set("prefix_list_ids", pl)

	if len(rule.UserIdGroupPairs) > 0 {
		s := rule.UserIdGroupPairs[0]

		if isVPC {
			d.Set("source_security_group_id", *s.GroupId)
		} else {
			d.Set("source_security_group_id", *s.GroupName)
		}
	}

	return nil
}

// Validates that either 'cidr_blocks', 'ipv6_cidr_blocks', 'self', or 'source_security_group_id' is set
func validateAwsSecurityGroupRule(d *schema.ResourceData) error {
	_, blocksOk := d.GetOk("cidr_blocks")
	_, ipv6Ok := d.GetOk("ipv6_cidr_blocks")
	_, sourceOk := d.GetOk("source_security_group_id")
	_, selfOk := d.GetOk("self")
	_, prefixOk := d.GetOk("prefix_list_ids")
	if !blocksOk && !sourceOk && !selfOk && !prefixOk && !ipv6Ok {
		return fmt.Errorf(
			"One of ['cidr_blocks', 'ipv6_cidr_blocks', 'self', 'source_security_group_id', 'prefix_list_ids'] must be set to create an AWS Security Group Rule")
	}
	return nil
}
