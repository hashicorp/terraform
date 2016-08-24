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
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

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
			"accepter":  vpcPeeringConnectionOptionsSchema(),
			"requester": vpcPeeringConnectionOptionsSchema(),
			"tags":      tagsSchema(),
		},
	}
}

func resourceAwsVPCPeeringCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Create the vpc peering connection
	createOpts := &ec2.CreateVpcPeeringConnectionInput{
		PeerOwnerId: aws.String(d.Get("peer_owner_id").(string)),
		PeerVpcId:   aws.String(d.Get("peer_vpc_id").(string)),
		VpcId:       aws.String(d.Get("vpc_id").(string)),
	}
	log.Printf("[DEBUG] VPC Peering Create options: %#v", createOpts)

	resp, err := conn.CreateVpcPeeringConnection(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating VPC Peering Connection: %s", err)
	}

	// Get the ID and store it
	rt := resp.VpcPeeringConnection
	d.SetId(*rt.VpcPeeringConnectionId)
	log.Printf("[INFO] VPC Peering Connection ID: %s", d.Id())

	// Wait for the vpc peering connection to become available
	log.Printf("[DEBUG] Waiting for VPC Peering Connection (%s) to become available.", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  []string{"pending-acceptance"},
		Refresh: resourceAwsVPCPeeringConnectionStateRefreshFunc(conn, d.Id()),
		Timeout: 1 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for VPC Peering Connection (%s) to become available: %s",
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

	pc := pcRaw.(*ec2.VpcPeeringConnection)

	// The failed status is a status that we can assume just means the
	// connection is gone. Destruction isn't allowed, and it eventually
	// just "falls off" the console. See GH-2322
	if pc.Status != nil {
		status := map[string]bool{
			"deleted":  true,
			"deleting": true,
			"expired":  true,
			"failed":   true,
			"rejected": true,
		}
		if _, ok := status[*pc.Status.Code]; ok {
			log.Printf("[DEBUG] VPC Peering Connection (%s) in state (%s), removing.",
				d.Id(), *pc.Status.Code)
			d.SetId("")
			return nil
		}
	}
	log.Printf("[DEBUG] VPC Peering Connection response: %#v", pc)

	d.Set("accept_status", pc.Status.Code)
	d.Set("peer_owner_id", pc.AccepterVpcInfo.OwnerId)
	d.Set("peer_vpc_id", pc.AccepterVpcInfo.VpcId)
	d.Set("vpc_id", pc.RequesterVpcInfo.VpcId)

	// When the VPC Peering Connection is pending acceptance,
	// the details about accepter and/or requester peering
	// options would not be included in the response.
	if pc.AccepterVpcInfo != nil && pc.AccepterVpcInfo.PeeringOptions != nil {
		err := d.Set("accepter", flattenPeeringOptions(pc.AccepterVpcInfo.PeeringOptions))
		if err != nil {
			log.Printf("[ERR] Error setting VPC Peering connection accepter information: %s", err)
		}
	}

	if pc.RequesterVpcInfo != nil && pc.RequesterVpcInfo.PeeringOptions != nil {
		err := d.Set("requester", flattenPeeringOptions(pc.RequesterVpcInfo.PeeringOptions))
		if err != nil {
			log.Printf("[ERR] Error setting VPC Peering connection requester information: %s", err)
		}
	}

	err = d.Set("tags", tagsToMap(pc.Tags))
	if err != nil {
		return err
	}

	return nil
}

func resourceVPCPeeringConnectionAccept(conn *ec2.EC2, id string) (string, error) {
	log.Printf("[INFO] Accept VPC Peering Connection with ID: %s", id)

	req := &ec2.AcceptVpcPeeringConnectionInput{
		VpcPeeringConnectionId: aws.String(id),
	}

	resp, err := conn.AcceptVpcPeeringConnection(req)
	if err != nil {
		return "", err
	}
	pc := resp.VpcPeeringConnection

	return *pc.Status.Code, nil
}

func resourceVPCPeeringConnectionOptionsModify(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	modifyOpts := &ec2.ModifyVpcPeeringConnectionOptionsInput{
		VpcPeeringConnectionId: aws.String(d.Id()),
	}

	if v, ok := d.GetOk("accepter"); ok {
		if s := v.(*schema.Set); len(s.List()) > 0 {
			co := s.List()[0].(map[string]interface{})
			modifyOpts.AccepterPeeringConnectionOptions = expandPeeringOptions(co)
		}
	}

	if v, ok := d.GetOk("requester"); ok {
		if s := v.(*schema.Set); len(s.List()) > 0 {
			co := s.List()[0].(map[string]interface{})
			modifyOpts.RequesterPeeringConnectionOptions = expandPeeringOptions(co)
		}
	}

	log.Printf("[DEBUG] VPC Peering Connection modify options: %#v", modifyOpts)
	if _, err := conn.ModifyVpcPeeringConnectionOptions(modifyOpts); err != nil {
		return err
	}

	return nil
}

func resourceAwsVPCPeeringUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	if err := setTags(conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	pcRaw, _, err := resourceAwsVPCPeeringConnectionStateRefreshFunc(conn, d.Id())()
	if err != nil {
		return err
	}
	if pcRaw == nil {
		d.SetId("")
		return nil
	}
	pc := pcRaw.(*ec2.VpcPeeringConnection)

	if _, ok := d.GetOk("auto_accept"); ok {
		if pc.Status != nil && *pc.Status.Code == "pending-acceptance" {
			status, err := resourceVPCPeeringConnectionAccept(conn, d.Id())
			if err != nil {
				return err
			}
			log.Printf("[DEBUG] VPC Peering Connection accept status: %s", status)
		}
	}

	if d.HasChange("accepter") || d.HasChange("requester") {
		_, ok := d.GetOk("auto_accept")
		if !ok && pc.Status != nil && *pc.Status.Code != "active" {
			return fmt.Errorf("Unable to modify peering options. The VPC Peering Connection "+
				"%q is not active. Please set `auto_accept` attribute to `true`, "+
				"or activate VPC Peering Connection manually.", d.Id())
		}

		if err := resourceVPCPeeringConnectionOptionsModify(d, meta); err != nil {
			return fmt.Errorf("Error modifying VPC Peering Connection options: %s", err)
		}
	}

	return resourceAwsVPCPeeringRead(d, meta)
}

func resourceAwsVPCPeeringDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	_, err := conn.DeleteVpcPeeringConnection(
		&ec2.DeleteVpcPeeringConnectionInput{
			VpcPeeringConnectionId: aws.String(d.Id()),
		})

	return err
}

// resourceAwsVPCPeeringConnectionStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a VPCPeeringConnection.
func resourceAwsVPCPeeringConnectionStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {

		resp, err := conn.DescribeVpcPeeringConnections(&ec2.DescribeVpcPeeringConnectionsInput{
			VpcPeeringConnectionIds: []*string{aws.String(id)},
		})
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpcPeeringConnectionID.NotFound" {
				resp = nil
			} else {
				log.Printf("Error reading VPC Peering Connection details: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		pc := resp.VpcPeeringConnections[0]

		return pc, *pc.Status.Code, nil
	}
}

func vpcPeeringConnectionOptionsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Computed: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"allow_remote_vpc_dns_resolution": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				"allow_classic_link_to_remote_vpc": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				"allow_vpc_to_remote_classic_link": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
			},
		},
	}
}

func flattenPeeringOptions(options *ec2.VpcPeeringConnectionOptionsDescription) (results []map[string]interface{}) {
	m := make(map[string]interface{})

	if options.AllowDnsResolutionFromRemoteVpc != nil {
		m["allow_remote_vpc_dns_resolution"] = *options.AllowDnsResolutionFromRemoteVpc
	}

	if options.AllowEgressFromLocalClassicLinkToRemoteVpc != nil {
		m["allow_classic_link_to_remote_vpc"] = *options.AllowEgressFromLocalClassicLinkToRemoteVpc
	}

	if options.AllowEgressFromLocalVpcToRemoteClassicLink != nil {
		m["allow_vpc_to_remote_classic_link"] = *options.AllowEgressFromLocalVpcToRemoteClassicLink
	}

	results = append(results, m)
	return
}

func expandPeeringOptions(m map[string]interface{}) *ec2.PeeringConnectionOptionsRequest {
	r := &ec2.PeeringConnectionOptionsRequest{}

	if v, ok := m["allow_remote_vpc_dns_resolution"]; ok {
		r.AllowDnsResolutionFromRemoteVpc = aws.Bool(v.(bool))
	}

	if v, ok := m["allow_classic_link_to_remote_vpc"]; ok {
		r.AllowEgressFromLocalClassicLinkToRemoteVpc = aws.Bool(v.(bool))
	}

	if v, ok := m["allow_vpc_to_remote_classic_link"]; ok {
		r.AllowEgressFromLocalVpcToRemoteClassicLink = aws.Bool(v.(bool))
	}

	return r
}
