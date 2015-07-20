package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpcPeeringConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVPCPeeringCreate,
		Read:   resourceAwsVPCPeeringRead,
		Update: resourceAwsVPCPeeringUpdate,
		Delete: resourceAwsVPCPeeringDelete,

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

func resourceAwsVPCPeeringCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Create the vpc peering connection
	createOpts := &ec2.CreateVPCPeeringConnectionInput{
		PeerOwnerID: aws.String(d.Get("peer_owner_id").(string)),
		PeerVPCID:   aws.String(d.Get("peer_vpc_id").(string)),
		VPCID:       aws.String(d.Get("vpc_id").(string)),
	}
	log.Printf("[DEBUG] VPCPeeringCreate create config: %#v", createOpts)
	resp, err := conn.CreateVPCPeeringConnection(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating vpc peering connection: %s", err)
	}

	// Get the ID and store it
	rt := resp.VPCPeeringConnection
	d.SetId(*rt.VPCPeeringConnectionID)
	log.Printf("[INFO] VPC Peering Connection ID: %s", d.Id())

	// Wait for the vpc peering connection to become available
	log.Printf(
		"[DEBUG] Waiting for vpc peering connection (%s) to become available",
		d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  "pending-acceptance",
		Refresh: resourceAwsVPCPeeringConnectionStateRefreshFunc(conn, d.Id()),
		Timeout: 1 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for vpc peering (%s) to become available: %s",
			d.Id(), err)
	}

	return resourceAwsVPCPeeringUpdate(d, meta)
}

func resourceAwsVPCPeeringRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	pcRaw, _, err := resourceAwsVPCPeeringConnectionStateRefreshFunc(conn, d.Id())()
	if err != nil {
		return err
	}
	if pcRaw == nil {
		d.SetId("")
		return nil
	}

	pc := pcRaw.(*ec2.VPCPeeringConnection)

	// The failed status is a status that we can assume just means the
	// connection is gone. Destruction isn't allowed, and it eventually
	// just "falls off" the console. See GH-2322
	if *pc.Status.Code == "failed" {
		d.SetId("")
		return nil
	}

	d.Set("accept_status", *pc.Status.Code)
	d.Set("peer_owner_id", pc.AccepterVPCInfo.OwnerID)
	d.Set("peer_vpc_id", pc.AccepterVPCInfo.VPCID)
	d.Set("vpc_id", pc.RequesterVPCInfo.VPCID)
	d.Set("tags", tagsToMap(pc.Tags))

	return nil
}

func resourceVPCPeeringConnectionAccept(conn *ec2.EC2, id string) (string, error) {

	log.Printf("[INFO] Accept VPC Peering Connection with id: %s", id)

	req := &ec2.AcceptVPCPeeringConnectionInput{
		VPCPeeringConnectionID: aws.String(id),
	}

	resp, err := conn.AcceptVPCPeeringConnection(req)
	pc := resp.VPCPeeringConnection
	return *pc.Status.Code, err
}

func resourceAwsVPCPeeringUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	if err := setTags(conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	if _, ok := d.GetOk("auto_accept"); ok {

		pcRaw, _, err := resourceAwsVPCPeeringConnectionStateRefreshFunc(conn, d.Id())()

		if err != nil {
			return err
		}
		if pcRaw == nil {
			d.SetId("")
			return nil
		}
		pc := pcRaw.(*ec2.VPCPeeringConnection)

		if *pc.Status.Code == "pending-acceptance" {

			status, err := resourceVPCPeeringConnectionAccept(conn, d.Id())

			log.Printf(
				"[DEBUG] VPC Peering connection accept status %s",
				status)
			if err != nil {
				return err
			}
		}
	}

	return resourceAwsVPCPeeringRead(d, meta)
}

func resourceAwsVPCPeeringDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	_, err := conn.DeleteVPCPeeringConnection(
		&ec2.DeleteVPCPeeringConnectionInput{
			VPCPeeringConnectionID: aws.String(d.Id()),
		})
	return err
}

// resourceAwsVPCPeeringConnectionStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a VPCPeeringConnection.
func resourceAwsVPCPeeringConnectionStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {

		resp, err := conn.DescribeVPCPeeringConnections(&ec2.DescribeVPCPeeringConnectionsInput{
			VPCPeeringConnectionIDs: []*string{aws.String(id)},
		})
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpcPeeringConnectionID.NotFound" {
				resp = nil
			} else {
				log.Printf("Error on VPCPeeringConnectionStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		pc := resp.VPCPeeringConnections[0]

		return pc, *pc.Status.Code, nil
	}
}
