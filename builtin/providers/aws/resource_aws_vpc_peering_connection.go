package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpcPeeringConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVpcPeeringCreate,
		Read:   resourceAwsVpcPeeringRead,
		Update: resourceAwsVpcPeeringUpdate,
		Delete: resourceAwsVpcPeeringDelete,

		Schema: map[string]*schema.Schema{
			"peer_owner_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("AWS_ACCOUNT_ID", nil),
			},
			"peer_vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsVpcPeeringCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Create the vpc peering connection
	createOpts := &ec2.CreateVPCPeeringConnectionInput{
		PeerOwnerID: aws.String(d.Get("peer_owner_id").(string)),
		PeerVPCID:   aws.String(d.Get("peer_vpc_id").(string)),
		VPCID:       aws.String(d.Get("vpc_id").(string)),
	}
	log.Printf("[DEBUG] VpcPeeringCreate create config: %#v", createOpts)
	resp, err := conn.CreateVPCPeeringConnection(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating vpc peering connection: %s", err)
	}

	// Get the ID and store it
	rt := resp.VPCPeeringConnection
	d.SetId(*rt.VPCPeeringConnectionID)
	log.Printf("[INFO] Vpc Peering Connection ID: %s", d.Id())

	// Wait for the vpc peering connection to become available
	log.Printf(
		"[DEBUG] Waiting for vpc peering connection (%s) to become available",
		d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  "ready",
		Refresh: resourceAwsVpcPeeringConnectionStateRefreshFunc(conn, d.Id()),
		Timeout: 1 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for vpc peering (%s) to become available: %s",
			d.Id(), err)
	}

	return nil
}

func resourceAwsVpcPeeringRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	pcRaw, _, err := resourceAwsVpcPeeringConnectionStateRefreshFunc(conn, d.Id())()
	if err != nil {
		return err
	}
	if pcRaw == nil {
		d.SetId("")
		return nil
	}

	pc := pcRaw.(*ec2.VPCPeeringConnection)

	d.Set("peer_owner_id", pc.AccepterVPCInfo.OwnerID)
	d.Set("peer_vpc_id", pc.AccepterVPCInfo.VPCID)
	d.Set("vpc_id", pc.RequesterVPCInfo.VPCID)
	d.Set("tags", tagsToMapSDK(pc.Tags))

	return nil
}

func resourceAwsVpcPeeringUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	if err := setTagsSDK(conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	return resourceAwsRouteTableRead(d, meta)
}

func resourceAwsVpcPeeringDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	_, err := conn.DeleteVPCPeeringConnection(
		&ec2.DeleteVPCPeeringConnectionInput{
			VPCPeeringConnectionID: aws.String(d.Id()),
		})
	return err
}

// resourceAwsVpcPeeringConnectionStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a VpcPeeringConnection.
func resourceAwsVpcPeeringConnectionStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {

		resp, err := conn.DescribeVPCPeeringConnections(&ec2.DescribeVPCPeeringConnectionsInput{
			VPCPeeringConnectionIDs: []*string{aws.String(id)},
		})
		if err != nil {
			if ec2err, ok := err.(aws.APIError); ok && ec2err.Code == "InvalidVpcPeeringConnectionID.NotFound" {
				resp = nil
			} else {
				log.Printf("Error on VpcPeeringConnectionStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		pc := resp.VPCPeeringConnections[0]

		return pc, "ready", nil
	}
}
