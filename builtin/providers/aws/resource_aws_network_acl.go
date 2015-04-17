package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsNetworkAcl() *schema.Resource {

	return &schema.Resource{
		Create: resourceAwsNetworkAclCreate,
		Read:   resourceAwsNetworkAclRead,
		Delete: resourceAwsNetworkAclDelete,
		Update: resourceAwsNetworkAclUpdate,

		Schema: map[string]*schema.Schema{
			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Computed: false,
			},
			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: false,
			},
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
					},
				},
				Set: resourceAwsNetworkAclEntryHash,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsNetworkAclCreate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).ec2conn

	// Create the Network Acl
	createOpts := &ec2.CreateNetworkACLInput{
		VPCID: aws.String(d.Get("vpc_id").(string)),
	}

	log.Printf("[DEBUG] Network Acl create config: %#v", createOpts)
	resp, err := conn.CreateNetworkACL(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating network acl: %s", err)
	}

	// Get the ID and store it
	networkAcl := resp.NetworkACL
	d.SetId(*networkAcl.NetworkACLID)
	log.Printf("[INFO] Network Acl ID: %s", *networkAcl.NetworkACLID)

	// Update rules and subnet association once acl is created
	return resourceAwsNetworkAclUpdate(d, meta)
}

func resourceAwsNetworkAclRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	resp, err := conn.DescribeNetworkACLs(&ec2.DescribeNetworkACLsInput{
		NetworkACLIDs: []*string{aws.String(d.Id())},
	})

	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}

	networkAcl := resp.NetworkACLs[0]
	var ingressEntries []*ec2.NetworkACLEntry
	var egressEntries []*ec2.NetworkACLEntry

	// separate the ingress and egress rules
	for _, e := range networkAcl.Entries {
		if *e.Egress == true {
			egressEntries = append(egressEntries, e)
		} else {
			ingressEntries = append(ingressEntries, e)
		}
	}

	d.Set("vpc_id", networkAcl.VPCID)
	d.Set("ingress", ingressEntries)
	d.Set("egress", egressEntries)
	d.Set("tags", tagsToMapSDK(networkAcl.Tags))

	return nil
}

func resourceAwsNetworkAclUpdate(d *schema.ResourceData, meta interface{}) error {
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

	if d.HasChange("subnet_id") {

		//associate new subnet with the acl.
		_, n := d.GetChange("subnet_id")
		newSubnet := n.(string)
		association, err := findNetworkAclAssociation(newSubnet, conn)
		if err != nil {
			return fmt.Errorf("Failed to update acl %s with subnet %s: %s", d.Id(), newSubnet, err)
		}
		_, err = conn.ReplaceNetworkACLAssociation(&ec2.ReplaceNetworkACLAssociationInput{
			AssociationID: association.NetworkACLAssociationID,
			NetworkACLID:  aws.String(d.Id()),
		})
		if err != nil {
			return err
		}
	}

	if err := setTagsSDK(conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)
	return resourceAwsNetworkAclRead(d, meta)
}

func updateNetworkAclEntries(d *schema.ResourceData, entryType string, conn *ec2.EC2) error {

	o, n := d.GetChange(entryType)

	if o == nil {
		o = new(schema.Set)
	}
	if n == nil {
		n = new(schema.Set)
	}

	os := o.(*schema.Set)
	ns := n.(*schema.Set)

	toBeDeleted, err := expandNetworkAclEntries(os.Difference(ns).List(), entryType)
	if err != nil {
		return err
	}
	for _, remove := range toBeDeleted {
		// Delete old Acl
		_, err := conn.DeleteNetworkACLEntry(&ec2.DeleteNetworkACLEntryInput{
			NetworkACLID: aws.String(d.Id()),
			RuleNumber:   remove.RuleNumber,
			Egress:       remove.Egress,
		})
		if err != nil {
			return fmt.Errorf("Error deleting %s entry: %s", entryType, err)
		}
	}

	toBeCreated, err := expandNetworkAclEntries(ns.Difference(os).List(), entryType)
	if err != nil {
		return err
	}
	for _, add := range toBeCreated {
		// Add new Acl entry
		_, err := conn.CreateNetworkACLEntry(&ec2.CreateNetworkACLEntryInput{
			NetworkACLID: aws.String(d.Id()),
			CIDRBlock:    add.CIDRBlock,
			Egress:       add.Egress,
			PortRange:    add.PortRange,
			Protocol:     add.Protocol,
			RuleAction:   add.RuleAction,
			RuleNumber:   add.RuleNumber,
		})
		if err != nil {
			return fmt.Errorf("Error creating %s entry: %s", entryType, err)
		}
	}
	return nil
}

func resourceAwsNetworkAclDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[INFO] Deleting Network Acl: %s", d.Id())
	return resource.Retry(5*time.Minute, func() error {
		_, err := conn.DeleteNetworkACL(&ec2.DeleteNetworkACLInput{
			NetworkACLID: aws.String(d.Id()),
		})
		if err != nil {
			ec2err := err.(aws.APIError)
			switch ec2err.Code {
			case "InvalidNetworkAclID.NotFound":
				return nil
			case "DependencyViolation":
				// In case of dependency violation, we remove the association between subnet and network acl.
				// This means the subnet is attached to default acl of vpc.
				association, err := findNetworkAclAssociation(d.Get("subnet_id").(string), conn)
				if err != nil {
					return fmt.Errorf("Dependency violation: Cannot delete acl %s: %s", d.Id(), err)
				}
				defaultAcl, err := getDefaultNetworkAcl(d.Get("vpc_id").(string), conn)
				if err != nil {
					return fmt.Errorf("Dependency violation: Cannot delete acl %s: %s", d.Id(), err)
				}
				_, err = conn.ReplaceNetworkACLAssociation(&ec2.ReplaceNetworkACLAssociationInput{
					AssociationID: association.NetworkACLAssociationID,
					NetworkACLID:  defaultAcl.NetworkACLID,
				})
				return resource.RetryError{Err: err}
			default:
				// Any other error, we want to quit the retry loop immediately
				return resource.RetryError{Err: err}
			}
		}
		log.Printf("[Info] Deleted network ACL %s successfully", d.Id())
		return nil
	})
}

func resourceAwsNetworkAclEntryHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["from_port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["to_port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["rule_no"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["action"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cidr_block"].(string)))

	if v, ok := m["ssl_certificate_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func getDefaultNetworkAcl(vpc_id string, conn *ec2.EC2) (defaultAcl *ec2.NetworkACL, err error) {
	resp, err := conn.DescribeNetworkACLs(&ec2.DescribeNetworkACLsInput{
		NetworkACLIDs: []*string{},
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("default"),
				Values: []*string{aws.String("true")},
			},
			&ec2.Filter{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpc_id)},
			},
		},
	})

	if err != nil {
		return nil, err
	}
	return resp.NetworkACLs[0], nil
}

func findNetworkAclAssociation(subnetId string, conn *ec2.EC2) (networkAclAssociation *ec2.NetworkACLAssociation, err error) {
	resp, err := conn.DescribeNetworkACLs(&ec2.DescribeNetworkACLsInput{
		NetworkACLIDs: []*string{},
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("association.subnet-id"),
				Values: []*string{aws.String(subnetId)},
			},
		},
	})

	if err != nil {
		return nil, err
	}
	for _, association := range resp.NetworkACLs[0].Associations {
		if *association.SubnetID == subnetId {
			return association, nil
		}
	}
	return nil, fmt.Errorf("could not find association for subnet %s ", subnetId)
}
