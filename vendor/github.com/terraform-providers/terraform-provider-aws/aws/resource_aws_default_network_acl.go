package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

// ACL Network ACLs all contain explicit deny-all rules that cannot be
// destroyed or changed by users. This rules are numbered very high to be a
// catch-all.
// See http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_ACLs.html#default-network-acl
const (
	awsDefaultAclRuleNumberIpv4 = 32767
	awsDefaultAclRuleNumberIpv6 = 32768
)

func resourceAwsDefaultNetworkAcl() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDefaultNetworkAclCreate,
		// We reuse aws_network_acl's read method, the operations are the same
		Read:   resourceAwsNetworkAclRead,
		Delete: resourceAwsDefaultNetworkAclDelete,
		Update: resourceAwsDefaultNetworkAclUpdate,

		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"default_network_acl_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Computed: false,
			},
			// We want explicit management of Subnets here, so we do not allow them to be
			// computed. Instead, an empty config will enforce just that; removal of the
			// any Subnets that have been assigned to the Default Network ACL. Because we
			// can't actually remove them, this will be a continual plan until the
			// Subnets are themselves destroyed or reassigned to a different Network
			// ACL
			"subnet_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			// We want explicit management of Rules here, so we do not allow them to be
			// computed. Instead, an empty config will enforce just that; removal of the
			// rules
			"ingress": {
				Type:     schema.TypeSet,
				Required: false,
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
						"rule_no": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"action": {
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
						},
						"cidr_block": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ipv6_cidr_block": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"icmp_type": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"icmp_code": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
				Set: resourceAwsNetworkAclEntryHash,
			},
			"egress": {
				Type:     schema.TypeSet,
				Required: false,
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
						"rule_no": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"action": {
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
						},
						"cidr_block": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ipv6_cidr_block": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"icmp_type": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"icmp_code": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
				Set: resourceAwsNetworkAclEntryHash,
			},

			"tags": tagsSchema(),

			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsDefaultNetworkAclCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(d.Get("default_network_acl_id").(string))

	// revoke all default and pre-existing rules on the default network acl.
	// In the UPDATE method, we'll apply only the rules in the configuration.
	log.Printf("[DEBUG] Revoking default ingress and egress rules for Default Network ACL for %s", d.Id())
	err := revokeAllNetworkACLEntries(d.Id(), meta)
	if err != nil {
		return err
	}

	return resourceAwsDefaultNetworkAclUpdate(d, meta)
}

func resourceAwsDefaultNetworkAclUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	d.Partial(true)

	if d.HasChange("ingress") {
		err := updateNetworkAclEntries(d, "ingress", conn)
		if err != nil {
			return err
		}
	}

	if d.HasChange("egress") {
		err := updateNetworkAclEntries(d, "egress", conn)
		if err != nil {
			return err
		}
	}

	if d.HasChange("subnet_ids") {
		o, n := d.GetChange("subnet_ids")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		remove := os.Difference(ns).List()
		add := ns.Difference(os).List()

		if len(remove) > 0 {
			//
			// NO-OP
			//
			// Subnets *must* belong to a Network ACL. Subnets are not "removed" from
			// Network ACLs, instead their association is replaced. In a normal
			// Network ACL, any removal of a Subnet is done by replacing the
			// Subnet/ACL association with an association between the Subnet and the
			// Default Network ACL. Because we're managing the default here, we cannot
			// do that, so we simply log a NO-OP. In order to remove the Subnet here,
			// it must be destroyed, or assigned to different Network ACL. Those
			// operations are not handled here
			log.Printf("[WARN] Cannot remove subnets from the Default Network ACL. They must be re-assigned or destroyed")
		}

		if len(add) > 0 {
			for _, a := range add {
				association, err := findNetworkAclAssociation(a.(string), conn)
				if err != nil {
					return fmt.Errorf("Failed to find acl association: acl %s with subnet %s: %s", d.Id(), a, err)
				}
				log.Printf("[DEBUG] Updating Network Association for Default Network ACL (%s) and Subnet (%s)", d.Id(), a.(string))
				_, err = conn.ReplaceNetworkAclAssociation(&ec2.ReplaceNetworkAclAssociationInput{
					AssociationId: association.NetworkAclAssociationId,
					NetworkAclId:  aws.String(d.Id()),
				})
				if err != nil {
					return err
				}
			}
		}
	}

	if err := setTags(conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)
	// Re-use the exiting Network ACL Resources READ method
	return resourceAwsNetworkAclRead(d, meta)
}

func resourceAwsDefaultNetworkAclDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[WARN] Cannot destroy Default Network ACL. Terraform will remove this resource from the state file, however resources may remain.")
	return nil
}

// revokeAllNetworkACLEntries revoke all ingress and egress rules that the Default
// Network ACL currently has
func revokeAllNetworkACLEntries(netaclId string, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	resp, err := conn.DescribeNetworkAcls(&ec2.DescribeNetworkAclsInput{
		NetworkAclIds: []*string{aws.String(netaclId)},
	})

	if err != nil {
		log.Printf("[DEBUG] Error looking up Network ACL: %s", err)
		return err
	}

	if resp == nil {
		return fmt.Errorf("Error looking up Default Network ACL Entries: No results")
	}

	networkAcl := resp.NetworkAcls[0]
	for _, e := range networkAcl.Entries {
		// Skip the default rules added by AWS. They can be neither
		// configured or deleted by users. See http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_ACLs.html#default-network-acl
		if *e.RuleNumber == awsDefaultAclRuleNumberIpv4 ||
			*e.RuleNumber == awsDefaultAclRuleNumberIpv6 {
			continue
		}

		// track if this is an egress or ingress rule, for logging purposes
		rt := "ingress"
		if *e.Egress == true {
			rt = "egress"
		}

		log.Printf("[DEBUG] Destroying Network ACL (%s) Entry number (%d)", rt, int(*e.RuleNumber))
		_, err := conn.DeleteNetworkAclEntry(&ec2.DeleteNetworkAclEntryInput{
			NetworkAclId: aws.String(netaclId),
			RuleNumber:   e.RuleNumber,
			Egress:       e.Egress,
		})
		if err != nil {
			return fmt.Errorf("Error deleting entry (%s): %s", e, err)
		}
	}

	return nil
}
