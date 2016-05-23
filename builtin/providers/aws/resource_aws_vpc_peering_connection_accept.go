package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpcPeeringConnectionAccept() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVPCPeeringAcceptCreate,
		Read:   resourceAwsVPCPeeringAcceptRead,
		Delete: resourceAwsVPCPeeringAcceptDelete,

		Schema: map[string]*schema.Schema{
			"peering_connection_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"accept_status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsVPCPeeringAcceptCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	if cur, ok := d.Get("accept_status").(string); ok && cur == ec2.VpcPeeringConnectionStateReasonCodeActive {
		// already accepted
		return nil
	}

	status, err := resourceVPCPeeringConnectionAccept(conn, d.Id())
	if err != nil {
		return err
	}
	d.Set("accept_status", status)

	// TODO: should we poll until this resolves? VpcPeeringConnectionStateReasonCodePendingAcceptance

	if status != ec2.VpcPeeringConnectionStateReasonCodeActive {
		return fmt.Errorf("Error accepting connection, state: %s", status)
	}
	return nil
}

func resourceAwsVPCPeeringAcceptRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	_, status, err := resourceAwsVPCPeeringConnectionStateRefreshFunc(conn, d.Id())()
	if err != nil {
		return err
	}
	d.Set("accept_status", status)
	d.SetId(d.Get("peering_connection_id").(string))
	return nil
}

func resourceAwsVPCPeeringAcceptDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
