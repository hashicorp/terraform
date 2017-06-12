package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSecurityGroupRules() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSecurityGroupRulesCreate,
		Read:   resourceAwsSecurityGroupRulesRead,
		Update: resourceAwsSecurityGroupRulesUpdate,
		Delete: resourceAwsSecurityGroupRulesDelete,

		Schema: map[string]*schema.Schema{
			"security_group_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ingress": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from_port": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"to_port": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"protocol": {
							Type:      schema.TypeString,
							Required:  true,
							StateFunc: protocolStateFunc,
						},

						"cidr_blocks": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validateCIDRNetworkAddress,
							},
						},

						"ipv6_cidr_blocks": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validateCIDRNetworkAddress,
							},
						},

						"security_groups": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"self": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
				Set: resourceAwsSecurityGroupRuleHash,
			},

			"egress": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from_port": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"to_port": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"protocol": {
							Type:      schema.TypeString,
							Required:  true,
							StateFunc: protocolStateFunc,
						},

						"cidr_blocks": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validateCIDRNetworkAddress,
							},
						},

						"ipv6_cidr_blocks": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validateCIDRNetworkAddress,
							},
						},

						"prefix_list_ids": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"security_groups": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"self": {
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
	return resourceAwsSecurityGroupRulesUpdate(d, meta)
}

func resourceAwsSecurityGroupRulesRead(d *schema.ResourceData, meta interface{}) error {
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

	remoteIngressRules := resourceAwsSecurityGroupIPPermGather(d.Id(), sg.IpPermissions, sg.OwnerId)
	remoteEgressRules := resourceAwsSecurityGroupIPPermGather(d.Id(), sg.IpPermissionsEgress, sg.OwnerId)

	localIngressRules := d.Get("ingress").(*schema.Set).List()
	localEgressRules := d.Get("egress").(*schema.Set).List()

	// Loop through the local state of rules, doing a match against the remote
	// ruleSet we built above.
	ingressRules := matchRules("ingress", localIngressRules, remoteIngressRules)
	egressRules := matchRules("egress", localEgressRules, remoteEgressRules)

	if err := d.Set("ingress", ingressRules); err != nil {
		log.Printf("[WARN] Error setting Ingress rule set for (%s): %s", d.Id(), err)
	}

	if err := d.Set("egress", egressRules); err != nil {
		log.Printf("[WARN] Error setting Egress rule set for (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceAwsSecurityGroupRulesUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	id := d.Get("security_group_id").(string)

	awsMutexKV.Lock(id)
	defer awsMutexKV.Unlock(id)

	sgRaw, _, err := SGStateRefreshFunc(conn, id)()
	if err != nil {
		return err
	}
	if sgRaw == nil {
		d.SetId("")
		return nil
	}

	d.SetId(id)

	group := sgRaw.(*ec2.SecurityGroup)
	isVPC := group.VpcId != nil && *group.VpcId != ""

	err = resourceAwsSecurityGroupUpdateRules(d, "ingress", meta, group)
	if err != nil {
		return err
	}

	if isVPC {
		err = resourceAwsSecurityGroupUpdateRules(d, "egress", meta, group)
		if err != nil {
			return err
		}
	}

	return resourceAwsSecurityGroupRulesRead(d, meta)
}

func resourceAwsSecurityGroupRulesDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	id := d.Get("security_group_id").(string)

	awsMutexKV.Lock(id)
	defer awsMutexKV.Unlock(id)

	log.Printf("[DEBUG] Security Group Rules destroy: %v", id)

	sgRaw, _, err := SGStateRefreshFunc(conn, id)()
	if err != nil {
		return err
	}
	if sgRaw == nil {
		return nil
	}

	group := sgRaw.(*ec2.SecurityGroup)

	ingress, err := expandIPPerms(group, d.Get("ingress").(*schema.Set).List())
	if err != nil {
		return err
	}
	egress, err := expandIPPerms(group, d.Get("egress").(*schema.Set).List())
	if err != nil {
		return err
	}

	if len(ingress) > 0 || len(egress) > 0 {
		conn := meta.(*AWSClient).ec2conn

		var err error
		if len(ingress) > 0 {
			log.Printf("[DEBUG] Revoking security group %#v %s rule: %#v",
				group, "ingress", ingress)

			req := &ec2.RevokeSecurityGroupIngressInput{
				GroupId:       group.GroupId,
				IpPermissions: ingress,
			}
			if group.VpcId == nil || *group.VpcId == "" {
				req.GroupId = nil
				req.GroupName = group.GroupName
			}
			_, err = conn.RevokeSecurityGroupIngress(req)
			if err != nil {
				return fmt.Errorf(
					"Error revoking security group %s rules: %s",
					"ingress", err)
			}
		}

		if len(egress) > 0 {
			log.Printf("[DEBUG] Revoking security group %#v %s rule: %#v",
				group, "egress", egress)

			req := &ec2.RevokeSecurityGroupEgressInput{
				GroupId:       group.GroupId,
				IpPermissions: egress,
			}
			_, err = conn.RevokeSecurityGroupEgress(req)
			if err != nil {
				return fmt.Errorf(
					"Error revoking security group %s rules: %s",
					"egress", err)
			}
		}
	}

	d.SetId("")

	return nil
}
