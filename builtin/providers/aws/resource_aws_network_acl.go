package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/ec2"
)

func resourceAwsNetworkAcl() *schema.Resource {

	return &schema.Resource{
		Create: 		resourceAwsNetworkAclCreate,
		Read:   		resourceAwsNetworkAclRead,
		Delete:   		resourceAwsNetworkAclDelete,
		Update: 		resourceAwsNetworkAclUpdate,

		Schema: map[string]*schema.Schema{
			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
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
	return nil 
	// resource_aws_subnet_update_state(s, subnetRaw.(*ec2.Subnet))
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

	d.Set("vpc_id", networkAcl.VpcId)

	return nil
}


func resourceAwsNetworkAclUpdate(d *schema.ResourceData, meta interface{}) error {

	return resourceAwsNetworkAclRead(d, meta)
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
