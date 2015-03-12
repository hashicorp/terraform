package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/ec2"
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
			"auto_accept": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"accept_status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsVpcPeeringCreate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).awsEC2conn

	// Create the vpc peering connection
	createOpts := &ec2.CreateVPCPeeringConnectionRequest{
		PeerOwnerID: aws.String(d.Get("peer_owner_id").(string)),
		PeerVPCID:   aws.String(d.Get("peer_vpc_id").(string)),
		VPCID:       aws.String(d.Get("vpc_id").(string)),
	}
	log.Printf("[DEBUG] VpcPeeringCreate create config: %#v", createOpts)
	resp, err := ec2conn.CreateVPCPeeringConnection(createOpts)
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
		Refresh: resourceAwsVpcPeeringConnectionStateRefreshFunc(ec2conn, d.Id()),
		Timeout: 1 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for vpc peering (%s) to become available: %s",
			d.Id(), err)
	}

	return resourceAwsVpcPeeringUpdate(d, meta)
}

func resourceAwsVpcPeeringRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).awsEC2conn
	pcRaw, _, err := resourceAwsVpcPeeringConnectionStateRefreshFunc(ec2conn, d.Id())()
	if err != nil {
		return err
	}
	if pcRaw == nil {
		d.SetId("")
		return nil
	}

	pc := pcRaw.(*ec2.VPCPeeringConnection)

	code := *pc.Status.Code
	if _, ok := d.GetOk("auto_accept"); ok {
		updatedCode, err := resourceVpcPeeringConnectionAccept(ec2conn, pc, d.Id())
		if err != nil {
			return fmt.Errorf("Error accepting vpc peering connection: %s", err)
		}

		code = updatedCode
	}

	d.Set("accept_status", code)

	d.Set("peer_owner_id", pc.AccepterVPCInfo.OwnerID)
	d.Set("peer_vpc_id", pc.AccepterVPCInfo.VPCID)
	d.Set("vpc_id", pc.RequesterVPCInfo.VPCID)
	d.Set("tags", tagsToMapSDK(pc.Tags))

	return nil
}

func resourceVpcPeeringConnectionAccept(conn *ec2.EC2, oldPc *ec2.VPCPeeringConnection, id string) (string, error) {

	if *oldPc.Status.Code == "pending-acceptance" {
		log.Printf("[INFO] Accept Vpc Peering Connection with id: %s", id)

		req := &ec2.AcceptVPCPeeringConnectionRequest{
			VPCPeeringConnectionID: aws.String(id),
		}
		_, err := conn.AcceptVPCPeeringConnection(req)

		pcRaw, _, err := resourceAwsVpcPeeringConnectionStateRefreshFunc(conn, id)()
		pc := pcRaw.(*ec2.VPCPeeringConnection)
		return *pc.Status.Code, err
	}

	return *oldPc.Status.Code, nil
}

func resourceAwsVpcPeeringUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).awsEC2conn

	if err := setTagsSDK(ec2conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	return resourceAwsVpcPeeringRead(d, meta)
}

func resourceAwsVpcPeeringDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).awsEC2conn

	_, err := ec2conn.DeleteVPCPeeringConnection(
		&ec2.DeleteVPCPeeringConnectionRequest{
			VPCPeeringConnectionID: aws.String(d.Id()),
		})
	return err
}

// resourceAwsVpcPeeringConnectionStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a VpcPeeringConnection.
func resourceAwsVpcPeeringConnectionStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {

		resp, err := conn.DescribeVPCPeeringConnections(&ec2.DescribeVPCPeeringConnectionsRequest{
			VPCPeeringConnectionIDs: []string{id},
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

		pc := &resp.VPCPeeringConnections[0]

		return pc, "ready", nil
	}
}
