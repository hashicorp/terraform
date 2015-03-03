package aws

import (
	//"bytes"
	//"crypto/sha1"
	//"encoding/hex"
	"fmt"
	"log"
	//"strconv"
	//"strings"
	//"time"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/aws-sdk-go/gen/ec2"
)

func resourceAwsNetworkInterface() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsNetworkInterfaceCreate,
		Read:   resourceAwsNetworkInterfaceRead,
		Update: resourceAwsNetworkInterfaceUpdate,
		Delete: resourceAwsNetworkInterfaceDelete,

		Schema: map[string]*schema.Schema{			
			
			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,				
				ForceNew: true,
			},

			"private_ips": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,				
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"source_dest_check": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},			

			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,				
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
			
			"tags": tagsSchema(),			
		},
	}
}

func resourceAwsNetworkInterfaceCreate(d *schema.ResourceData, meta interface{}) error {
	
	ec2conn := meta.(*AWSClient).ec2conn2

		request := &ec2.CreateNetworkInterfaceRequest{
		Description:			aws.String("xxx"),
		Groups: 				expandStringList(d.Get("security_groups").(*schema.Set).List()),
		SubnetID:				aws.String(d.Get("subnet_id").(string)),
	}
	
	log.Printf("[DEBUG] Creating network interface")
	resp, err := ec2conn.CreateNetworkInterface(request)
	if err != nil {
		return fmt.Errorf("Error creating ENI: %s", err)
	}

	new_interface_id := *resp.NetworkInterface.NetworkInterfaceID
	d.SetId(new_interface_id)

	
	// chain to update here
	return nil

}

func resourceAwsNetworkInterfaceRead(d *schema.ResourceData, meta interface{}) error {

	ec2conn := meta.(*AWSClient).ec2conn2
	describe_network_interfaces_request := &ec2.DescribeNetworkInterfacesRequest{		
	}
	describeResp, err := ec2conn.DescribeNetworkInterfaces(describe_network_interfaces_request)

	if err != nil {
		return nil
	}
	if describeResp != nil {
		return nil
	}

	return nil
}

func resourceAwsNetworkInterfaceUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsNetworkInterfaceDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

// InstanceStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an EC2 instance.
func NetworkInterfaceStateRefreshFunc(conn *ec2.EC2, instanceID string) resource.StateRefreshFunc {
	return nil
}