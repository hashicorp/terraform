package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
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

	// Update rules and subnet association once acl is created
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

	// separate the ingress and egress rules
	for _, e := range networkAcl.EntrySet {
		if e.Egress == true {
			egressEntries = append(egressEntries, e)
		} else {
			ingressEntries = append(ingressEntries, e)
		}
	}

	d.Set("vpc_id", networkAcl.VpcId)
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

	if(d.HasChange("subnet_id")) {
		association, err := findNetworkAclAssociation(d.Get("subnet_id").(string), ec2conn)
		if(err != nil){
			return fmt.Errorf("Depedency voilation: Could find association: %s", d.Id(), err)
		}
		// change acl and subnet association if subnet_id has changed
		_, err = ec2conn.ReplaceNetworkAclAssociation(association.NetworkAclAssociationId, d.Id())
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
	fmt.Printf("New : %s", n)

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
		// Delete old Acl
		_, err := ec2conn.DeleteNetworkAclEntry(d.Id(), remove.RuleNumber, remove.Egress)
		if err != nil {
			return fmt.Errorf("Error deleting %s entry: %s", entryType, err)
		}
	}
	fmt.Printf("to be deleted %s", toBeDeleted)

	for _, add := range toBeCreated {
		// Add new Acl entry
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
	return resource.Retry(5*time.Minute, func() error {
		if _, err := ec2conn.DeleteNetworkAcl(d.Id()); err != nil {
			ec2err := err.(*ec2.Error)
			fmt.Printf("\n\n error code: %s \n", ec2err.Code)
			switch ec2err.Code {
				case "InvalidNetworkAclID.NotFound":
					return nil
				case "DependencyViolation":
					// In case of dependency violation, we remove the association between subnet and network acl. 
					// This means the subnet is attached to default acl of vpc.
					association, err := findNetworkAclAssociation(d.Get("subnet_id").(string), ec2conn)
					if(err != nil){
						return fmt.Errorf("Depedency voilation: Could find association: %s", d.Id(), err)
					}
					defaultAcl, err := getDefaultNetworkAcl(d.Get("vpc_id").(string), ec2conn)
					if(err != nil){
						return fmt.Errorf("Depedency voilation: Could not dissociate subnet from %s acl: %s", d.Id(), err)
					}
					_, err = ec2conn.ReplaceNetworkAclAssociation(association.NetworkAclAssociationId, defaultAcl.NetworkAclId)
					return err
				default:
					// Any other error, we want to quit the retry loop immediately
					return resource.RetryError{err}
			}
		}
		return nil
	})
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


func getDefaultNetworkAcl(vpc_id string, ec2conn *ec2.EC2)(defaultAcl *ec2.NetworkAcl, err error){
	filter := ec2.NewFilter()
	filter.Add("default", "true" )
	filter.Add("vpc-id", vpc_id )

	resp, err := ec2conn.NetworkAcls([]string{}, filter)

	if err != nil {
		return nil, err
	}
	return &resp.NetworkAcls[0], nil
	}

func findNetworkAclAssociation(subnet_id string,ec2conn *ec2.EC2)(networkAclAssociation *ec2.NetworkAclAssociation, err error){
	filter := ec2.NewFilter()
	filter.Add("association.subnet-id", subnet_id )
		
	resp, err := ec2conn.NetworkAcls([]string{}, filter)

	if err != nil {
		return nil, err
	}
	return &resp.NetworkAcls[0].AssociationSet[0], nil
}
