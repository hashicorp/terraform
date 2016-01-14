package aws

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSecurityGroupCreate,
		Read:   resourceAwsSecurityGroupRead,
		Update: resourceAwsSecurityGroupUpdate,
		Delete: resourceAwsSecurityGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 255 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 255 characters", k))
					}
					return
				},
			},

			"name_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 100 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 100 characters, name is limited to 255", k))
					}
					return
				},
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "Managed by Terraform",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 255 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 255 characters", k))
					}
					return
				},
			},

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"ingress": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"to_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"cidr_blocks": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"security_groups": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set: func(v interface{}) int {
								return hashcode.String(v.(string))
							},
						},

						"self": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
				Set: resourceAwsSecurityGroupRuleHash,
			},

			"egress": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"to_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"cidr_blocks": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"security_groups": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set: func(v interface{}) int {
								return hashcode.String(v.(string))
							},
						},

						"self": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
				Set: resourceAwsSecurityGroupRuleHash,
			},

			"owner_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsSecurityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	securityGroupOpts := &ec2.CreateSecurityGroupInput{}

	if v, ok := d.GetOk("vpc_id"); ok {
		securityGroupOpts.VpcId = aws.String(v.(string))
	}

	if v := d.Get("description"); v != nil {
		securityGroupOpts.Description = aws.String(v.(string))
	}

	var groupName string
	if v, ok := d.GetOk("name"); ok {
		groupName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		groupName = resource.PrefixedUniqueId(v.(string))
	} else {
		groupName = resource.UniqueId()
	}
	securityGroupOpts.GroupName = aws.String(groupName)

	var err error
	log.Printf(
		"[DEBUG] Security Group create configuration: %#v", securityGroupOpts)
	createResp, err := conn.CreateSecurityGroup(securityGroupOpts)
	if err != nil {
		return fmt.Errorf("Error creating Security Group: %s", err)
	}

	d.SetId(*createResp.GroupId)

	log.Printf("[INFO] Security Group ID: %s", d.Id())

	// Wait for the security group to truly exist
	log.Printf(
		"[DEBUG] Waiting for Security Group (%s) to exist",
		d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{""},
		Target:  "exists",
		Refresh: SGStateRefreshFunc(conn, d.Id()),
		Timeout: 1 * time.Minute,
	}

	resp, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Security Group (%s) to become available: %s",
			d.Id(), err)
	}

	// AWS defaults all Security Groups to have an ALLOW ALL egress rule. Here we
	// revoke that rule, so users don't unknowningly have/use it.
	group := resp.(*ec2.SecurityGroup)
	if group.VpcId != nil && *group.VpcId != "" {
		log.Printf("[DEBUG] Revoking default egress rule for Security Group for %s", d.Id())

		req := &ec2.RevokeSecurityGroupEgressInput{
			GroupId: createResp.GroupId,
			IpPermissions: []*ec2.IpPermission{
				&ec2.IpPermission{
					FromPort: aws.Int64(int64(0)),
					ToPort:   aws.Int64(int64(0)),
					IpRanges: []*ec2.IpRange{
						&ec2.IpRange{
							CidrIp: aws.String("0.0.0.0/0"),
						},
					},
					IpProtocol: aws.String("-1"),
				},
			},
		}

		if _, err = conn.RevokeSecurityGroupEgress(req); err != nil {
			return fmt.Errorf(
				"Error revoking default egress rule for Security Group (%s): %s",
				d.Id(), err)
		}

	}

	return resourceAwsSecurityGroupUpdate(d, meta)
}

func resourceAwsSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	sgRaw, _, err := SGStateRefreshFunc(conn, d.Id())()
	if err != nil {
		return err
	}
	if sgRaw == nil {
		d.SetId("")
		return nil
	}

	sg := sgRaw.(*ec2.SecurityGroup)

	ingressRules := resourceAwsSecurityGroupIPPermGather(d, sg.IpPermissions)
	egressRules := resourceAwsSecurityGroupIPPermGather(d, sg.IpPermissionsEgress)

	d.Set("description", sg.Description)
	d.Set("name", sg.GroupName)
	d.Set("vpc_id", sg.VpcId)
	d.Set("owner_id", sg.OwnerId)
	d.Set("ingress", ingressRules)
	d.Set("egress", egressRules)
	d.Set("tags", tagsToMap(sg.Tags))
	return nil
}

func resourceAwsSecurityGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	sgRaw, _, err := SGStateRefreshFunc(conn, d.Id())()
	if err != nil {
		return err
	}
	if sgRaw == nil {
		d.SetId("")
		return nil
	}

	group := sgRaw.(*ec2.SecurityGroup)

	err = resourceAwsSecurityGroupUpdateRules(d, "ingress", meta, group)
	if err != nil {
		return err
	}

	if d.Get("vpc_id") != nil {
		err = resourceAwsSecurityGroupUpdateRules(d, "egress", meta, group)
		if err != nil {
			return err
		}
	}

	if err := setTags(conn, d); err != nil {
		return err
	}

	d.SetPartial("tags")

	return resourceAwsSecurityGroupRead(d, meta)
}

func resourceAwsSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Security Group destroy: %v", d.Id())

	return resource.Retry(5*time.Minute, func() error {
		_, err := conn.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(d.Id()),
		})
		if err != nil {
			ec2err, ok := err.(awserr.Error)
			if !ok {
				return err
			}

			switch ec2err.Code() {
			case "InvalidGroup.NotFound":
				return nil
			case "DependencyViolation":
				// If it is a dependency violation, we want to retry
				return err
			default:
				// Any other error, we want to quit the retry loop immediately
				return resource.RetryError{Err: err}
			}
		}

		return nil
	})
}

func resourceAwsSecurityGroupRuleHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["from_port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["to_port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%t-", m["self"].(bool)))

	// We need to make sure to sort the strings below so that we always
	// generate the same hash code no matter what is in the set.
	if v, ok := m["cidr_blocks"]; ok {
		vs := v.([]interface{})
		s := make([]string, len(vs))
		for i, raw := range vs {
			s[i] = raw.(string)
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}
	if v, ok := m["security_groups"]; ok {
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

func resourceAwsSecurityGroupIPPermGather(d *schema.ResourceData, permissions []*ec2.IpPermission) []map[string]interface{} {
	ruleMap := make(map[string]map[string]interface{})
	for _, perm := range permissions {
		var fromPort, toPort int64
		if v := perm.FromPort; v != nil {
			fromPort = *v
		}
		if v := perm.ToPort; v != nil {
			toPort = *v
		}

		k := fmt.Sprintf("%s-%d-%d", *perm.IpProtocol, fromPort, toPort)
		m, ok := ruleMap[k]
		if !ok {
			m = make(map[string]interface{})
			ruleMap[k] = m
		}

		m["from_port"] = fromPort
		m["to_port"] = toPort
		m["protocol"] = *perm.IpProtocol

		if len(perm.IpRanges) > 0 {
			raw, ok := m["cidr_blocks"]
			if !ok {
				raw = make([]string, 0, len(perm.IpRanges))
			}
			list := raw.([]string)

			for _, ip := range perm.IpRanges {
				list = append(list, *ip.CidrIp)
			}

			m["cidr_blocks"] = list
		}

		var groups []string
		if len(perm.UserIdGroupPairs) > 0 {
			groups = flattenSecurityGroups(perm.UserIdGroupPairs)
		}
		for i, id := range groups {
			if id == d.Id() {
				groups[i], groups = groups[len(groups)-1], groups[:len(groups)-1]
				m["self"] = true
			}
		}

		if len(groups) > 0 {
			raw, ok := m["security_groups"]
			if !ok {
				raw = make([]string, 0, len(groups))
			}
			list := raw.([]string)

			list = append(list, groups...)
			m["security_groups"] = list
		}
	}
	rules := make([]map[string]interface{}, 0, len(ruleMap))
	for _, m := range ruleMap {
		rules = append(rules, m)
	}
	return rules
}

func resourceAwsSecurityGroupUpdateRules(
	d *schema.ResourceData, ruleset string,
	meta interface{}, group *ec2.SecurityGroup) error {

	if d.HasChange(ruleset) {
		o, n := d.GetChange(ruleset)
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		remove, err := expandIPPerms(group, os.Difference(ns).List())
		if err != nil {
			return err
		}
		add, err := expandIPPerms(group, ns.Difference(os).List())
		if err != nil {
			return err
		}

		// TODO: We need to handle partial state better in the in-between
		// in this update.

		// TODO: It'd be nicer to authorize before removing, but then we have
		// to deal with complicated unrolling to get individual CIDR blocks
		// to avoid authorizing already authorized sources. Removing before
		// adding is easier here, and Terraform should be fast enough to
		// not have service issues.

		if len(remove) > 0 || len(add) > 0 {
			conn := meta.(*AWSClient).ec2conn

			var err error
			if len(remove) > 0 {
				log.Printf("[DEBUG] Revoking security group %#v %s rule: %#v",
					group, ruleset, remove)

				if ruleset == "egress" {
					req := &ec2.RevokeSecurityGroupEgressInput{
						GroupId:       group.GroupId,
						IpPermissions: remove,
					}
					_, err = conn.RevokeSecurityGroupEgress(req)
				} else {
					req := &ec2.RevokeSecurityGroupIngressInput{
						GroupId:       group.GroupId,
						IpPermissions: remove,
					}
					_, err = conn.RevokeSecurityGroupIngress(req)
				}

				if err != nil {
					return fmt.Errorf(
						"Error authorizing security group %s rules: %s",
						ruleset, err)
				}
			}

			if len(add) > 0 {
				log.Printf("[DEBUG] Authorizing security group %#v %s rule: %#v",
					group, ruleset, add)
				// Authorize the new rules
				if ruleset == "egress" {
					req := &ec2.AuthorizeSecurityGroupEgressInput{
						GroupId:       group.GroupId,
						IpPermissions: add,
					}
					_, err = conn.AuthorizeSecurityGroupEgress(req)
				} else {
					req := &ec2.AuthorizeSecurityGroupIngressInput{
						GroupId:       group.GroupId,
						IpPermissions: add,
					}
					if group.VpcId == nil || *group.VpcId == "" {
						req.GroupId = nil
						req.GroupName = group.GroupName
					}

					_, err = conn.AuthorizeSecurityGroupIngress(req)
				}

				if err != nil {
					return fmt.Errorf(
						"Error authorizing security group %s rules: %s",
						ruleset, err)
				}
			}
		}
	}
	return nil
}

// SGStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a security group.
func SGStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		req := &ec2.DescribeSecurityGroupsInput{
			GroupIds: []*string{aws.String(id)},
		}
		resp, err := conn.DescribeSecurityGroups(req)
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok {
				if ec2err.Code() == "InvalidSecurityGroupID.NotFound" ||
					ec2err.Code() == "InvalidGroup.NotFound" {
					resp = nil
					err = nil
				}
			}

			if err != nil {
				log.Printf("Error on SGStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			return nil, "", nil
		}

		group := resp.SecurityGroups[0]
		return group, "exists", nil
	}
}
