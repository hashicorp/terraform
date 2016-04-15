package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDefaultNetworkAcl() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDefaultNetworkAclCreate,
		Read:   resourceAwsNetworkAclRead,
		Delete: resourceAwsDefaultNetworkAclDelete,
		Update: resourceAwsDefaultNetworkAclUpdate,

		Schema: map[string]*schema.Schema{
			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"default_network_acl_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Computed: false,
			},
			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			// We want explicit managment of Subnets here, so we do not allow them to be
			// computed. Instead, an empty config will enforce just that; removal of the
			// any Subnets that have been assigned to the Default Network ACL. Because we
			// can't actually remove them, this will be a continual plan
			"subnet_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			// We want explicit managment of Rules here, so we do not allow them to be
			// computed. Instead, an empty config will enforce just that; removal of the
			// rules
			"ingress": &schema.Schema{
				Type:     schema.TypeSet,
				Required: false,
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
						"rule_no": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"action": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"cidr_block": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"icmp_type": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"icmp_code": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
				Set: resourceAwsNetworkAclEntryHash,
			},
			"egress": &schema.Schema{
				Type:     schema.TypeSet,
				Required: false,
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
						"rule_no": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"action": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"cidr_block": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"icmp_type": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"icmp_code": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
				Set: resourceAwsNetworkAclEntryHash,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDefaultNetworkAclCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(d.Get("default_network_acl_id").(string))
	log.Printf("[DEBUG] Revoking ingress rules for Default Network ACL for %s", d.Id())
	err := revokeRulesForType(d.Id(), "ingress", meta)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Revoking egress rules for Default Network ACL for %s", d.Id())
	err = revokeRulesForType(d.Id(), "egress", meta)
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
			// Subnets *must* belong to a Network ACL. Subnets are not "remove" from
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
	d.SetId("")
	return nil
}

// revokeRulesForType will query the Network ACL for it's entries, and revoke
// any rule of the matching type.
func revokeRulesForType(netaclId, rType string, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	resp, err := conn.DescribeNetworkAcls(&ec2.DescribeNetworkAclsInput{
		NetworkAclIds: []*string{aws.String(netaclId)},
	})

	if err != nil {
		log.Printf("[DEBUG] Error looking up Network ACL: %s", err)
		return err
	}

	if resp == nil {
		return fmt.Errorf("[ERR] Error looking up Default Network ACL Entries: No results")
	}

	networkAcl := resp.NetworkAcls[0]
	for _, e := range networkAcl.Entries {
		// Skip the default rules added by AWS. They can be neither
		// configured or deleted by users.
		if *e.RuleNumber == 32767 {
			continue
		}

		// networkAcl.Entries contains a list of ACL Entries, with an Egress boolean
		// to indicate if they are ingress or egress. Match on that bool to make
		// sure we're removing the right kind of rule, instead of just all rules
		rt := "ingress"
		if *e.Egress == true {
			rt = "egress"
		}

		if rType != rt {
			continue
		}

		log.Printf("[DEBUG] Destroying Network ACL Entry number (%d) for type (%s)", int(*e.RuleNumber), rt)
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
