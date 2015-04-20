package aws

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/nevins-b/terraform/helper/resource"
)

func resourceAwsSecurityGroupRules() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSecurityGroupRulesCreate,
		Read:   resourceAwsSecurityGroupRulesRead,
		Update: resourceAwsSecurityGroupRulesUpdate,
		Delete: resourceAwsSecurityGroupRulesDelete,

		Schema: map[string]*schema.Schema{
			"security_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ingress": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
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
		},
	}
}

func resourceAwsSecurityGroupRulesCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating Rules for Security Group ID: %s", d.Get("security_group_id"))
	ec2conn := meta.(*AWSClient).ec2conn

	sgRaw, _, err := SGStateRefreshFunc(ec2conn, d.Get("security_group_id").(string))()
	if err != nil {
		return err
	}
	group := sgRaw.(*ec2.SecurityGroup)

	if group.VPCID != nil {
		ereq := &ec2.RevokeSecurityGroupEgressInput{
			GroupID:       group.GroupID,
			IPPermissions: group.IPPermissionsEgress,
		}
		_, err = ec2conn.RevokeSecurityGroupEgress(ereq)
		if err != nil {
			return err
		}
	}

	id := fmt.Sprintf("%s-rules", d.Get("security_group_id").(string))
	d.SetId(id)

	return resourceAwsSecurityGroupRulesUpdate(d, meta)
}

func resourceAwsSecurityGroupRulesDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Security Group Rules destroy: %v", d.Id())

	return resource.Retry(5*time.Minute, func() error {

		sgRaw, _, err := SGStateRefreshFunc(ec2conn, d.Get("security_group_id").(string))()
		if err != nil {
			return err
		}
		group := sgRaw.(*ec2.SecurityGroup)
		ereq := &ec2.RevokeSecurityGroupEgressInput{
			GroupID:       group.GroupID,
			IPPermissions: group.IPPermissionsEgress,
		}
		_, err = ec2conn.RevokeSecurityGroupEgress(ereq)

		ireq := &ec2.RevokeSecurityGroupIngressInput{
			GroupID:       group.GroupID,
			IPPermissions: group.IPPermissions,
		}
		_, err = ec2conn.RevokeSecurityGroupIngress(ireq)
		if err != nil {
			ec2err, ok := err.(aws.APIError)
			if !ok {
				return err
			}

			switch ec2err.Code {
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

func resourceAwsSecurityGroupRulesRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	sgRaw, _, err := SGStateRefreshFunc(ec2conn, d.Get("security_group_id").(string))()
	if err != nil {
		return err
	}
	if sgRaw == nil {
		d.SetId("")
		return nil
	}

	sg := sgRaw.(*ec2.SecurityGroup)

	ingressRules := resourceAwsSecurityGroupIPPermGather(d, sg.IPPermissions)
	egressRules := resourceAwsSecurityGroupIPPermGather(d, sg.IPPermissionsEgress)

	d.Set("ingress", ingressRules)
	d.Set("egress", egressRules)
	return nil
}

func resourceAwsSecurityGroupRulesUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	sgRaw, _, err := SGStateRefreshFunc(ec2conn, d.Get("security_group_id").(string))()
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

	if group.VPCID != nil {
		err = resourceAwsSecurityGroupUpdateRules(d, "egress", meta, group)
		if err != nil {
			return err
		}
	}

	return resourceAwsSecurityGroupRulesRead(d, meta)
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

func resourceAwsSecurityGroupIPPermGather(d *schema.ResourceData, permissions []*ec2.IPPermission) []map[string]interface{} {
	ruleMap := make(map[string]map[string]interface{})
	for _, perm := range permissions {
		var fromPort, toPort int64
		if v := perm.FromPort; v != nil {
			fromPort = *v
		}
		if v := perm.ToPort; v != nil {
			toPort = *v
		}

		k := fmt.Sprintf("%s-%d-%d", *perm.IPProtocol, fromPort, toPort)
		m, ok := ruleMap[k]
		if !ok {
			m = make(map[string]interface{})
			ruleMap[k] = m
		}

		m["from_port"] = fromPort
		m["to_port"] = toPort
		m["protocol"] = *perm.IPProtocol

		if len(perm.IPRanges) > 0 {
			raw, ok := m["cidr_blocks"]
			if !ok {
				raw = make([]string, 0, len(perm.IPRanges))
			}
			list := raw.([]string)

			for _, ip := range perm.IPRanges {
				list = append(list, *ip.CIDRIP)
			}

			m["cidr_blocks"] = list
		}

		var groups []string
		if len(perm.UserIDGroupPairs) > 0 {
			groups = flattenSecurityGroups(perm.UserIDGroupPairs)
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
		log.Printf("[DEBUG] Rule %v", m)
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

		remove := expandIPPerms(group, os.Difference(ns).List())
		add := expandIPPerms(group, ns.Difference(os).List())

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
						GroupID:       group.GroupID,
						IPPermissions: remove,
					}
					_, err = conn.RevokeSecurityGroupEgress(req)
				} else {
					req := &ec2.RevokeSecurityGroupIngressInput{
						GroupID:       group.GroupID,
						IPPermissions: remove,
					}
					_, err = conn.RevokeSecurityGroupIngress(req)
				}

				if err != nil {
					return fmt.Errorf(
						"Error revoking security group %s rule: %s",
						ruleset, err)
				}
			}

			if len(add) > 0 {
				log.Printf("[DEBUG] Authorizing security group %#v %s rule: %#v",
					group, ruleset, add)
				// Authorize the new rules
				if ruleset == "egress" {
					req := &ec2.AuthorizeSecurityGroupEgressInput{
						GroupID:       group.GroupID,
						IPPermissions: add,
					}
					_, err = conn.AuthorizeSecurityGroupEgress(req)
				} else {
					req := &ec2.AuthorizeSecurityGroupIngressInput{
						GroupID:       group.GroupID,
						IPPermissions: add,
					}
					if group.VPCID == nil || *group.VPCID == "" {
						req.GroupID = nil
						req.GroupName = group.GroupName
					}

					_, err = conn.AuthorizeSecurityGroupIngress(req)
				}

				if err != nil {
					return fmt.Errorf(
						"Error authorizing security group %s rule: %s",
						ruleset, err)
				}
			}
		}
	}
	return nil
}
