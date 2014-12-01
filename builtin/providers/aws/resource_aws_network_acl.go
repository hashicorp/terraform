package aws

import (
	"bytes"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/ec2"
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
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
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
		},
	}
}

func resourceAwsNetworkAclCreate(d *schema.ResourceData, meta interface{}) error {

	ec2conn := meta.(*AWSClient).ec2conn

	// Create the Network Acl
	createOpts := &ec2.CreateNetworkAcl{
		VpcId: d.Get("vpc_id").(string),
	}
	log.Printf("[DEBUG] Network Acl create config: %#v", createOpts)
	resp, err := ec2conn.CreateNetworkAcl(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating network acl: %s", err)
	}

	// Get the ID and store it
	networkAcl := &resp.NetworkAcl
	d.SetId(networkAcl.NetworkAclId)
	log.Printf("[INFO] Network Acl ID: %s", networkAcl.NetworkAclId)

	// Update our attributes and return
	// return nil
	return resourceAwsNetworkAclUpdate(d, meta)
}

func resourceAwsNetworkAclRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	resp, err := ec2conn.NetworkAcls([]string{d.Id()}, ec2.NewFilter())

	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}

	networkAcl := &resp.NetworkAcls[0]
	var ingressEntries []ec2.NetworkAclEntry
	var egressEntries []ec2.NetworkAclEntry

	d.Set("vpc_id", networkAcl.VpcId)

	for _, e := range networkAcl.EntrySet {
		if e.Egress == true {
			egressEntries = append(egressEntries, e)
		} else {
			ingressEntries = append(ingressEntries, e)
		}
	}
	fmt.Printf("appending ingress entries %s", ingressEntries)
	fmt.Printf("appending egress entries %s", egressEntries)

	d.Set("ingress", ingressEntries)
	d.Set("egress", egressEntries)

	return nil
}

func resourceAwsNetworkAclUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn
	d.Partial(true)

	if d.HasChange("ingress") {
		err := updateNetworkAclEntries(d, "ingress", ec2conn)
		if err != nil {
			return err
		}
	}

	if d.HasChange("egress") {
		err := updateNetworkAclEntries(d, "egress", ec2conn)
		if err != nil {
			return err
		}
	}

	d.Partial(false)
	return resourceAwsNetworkAclRead(d, meta)

}

func updateNetworkAclEntries(d *schema.ResourceData, entryType string, ec2conn *ec2.EC2) error {

	o, n := d.GetChange(entryType)
	fmt.Printf("Old : %s", o)
	fmt.Printf("Old : %s", n)

	if o == nil {
		o = new(schema.Set)
	}
	if n == nil {
		n = new(schema.Set)
	}

	os := o.(*schema.Set)
	ns := n.(*schema.Set)

	toBeDeleted := expandNetworkAclEntries(os.Difference(ns).List(), entryType)
	toBeCreated := expandNetworkAclEntries(ns.Difference(os).List(), entryType)
	fmt.Printf("to be created %s", toBeCreated)
	for _, remove := range toBeDeleted {
		// Revoke the old entry
		_, err := ec2conn.DeleteNetworkAclEntry(d.Id(), remove.RuleNumber, remove.Egress)
		if err != nil {
			return fmt.Errorf("Error deleting %s entry: %s", entryType, err)
		}
	}
	fmt.Printf("to be deleted %s", toBeDeleted)

	for _, add := range toBeCreated {
		// Authorize the new entry
		_, err := ec2conn.CreateNetworkAclEntry(d.Id(), &add)
		fmt.Printf("$$$$#### %s", err)
		if err != nil {
			return fmt.Errorf("Error creating %s entry: %s", entryType, err)
		}
	}
	return nil
}

func resourceAwsNetworkAclDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	log.Printf("[INFO] Deleting Network Acl: %s", d.Id())
	if _, err := ec2conn.DeleteNetworkAcl(d.Id()); err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidNetworkAclID.NotFound" {
			return nil
		}

		return fmt.Errorf("Error deleting network acl: %s", err)
	}

	return nil
}

func resourceAwsNetworkAclEntryHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["from_port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["to_port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["rule_no"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["action"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cidr_block"].(string)))

	if v, ok := m["ssl_certificate_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}
