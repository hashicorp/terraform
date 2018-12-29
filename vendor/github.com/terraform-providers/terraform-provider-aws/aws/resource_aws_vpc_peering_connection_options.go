package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpcPeeringConnectionOptions() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVpcPeeringConnectionOptionsCreate,
		Read:   resourceAwsVpcPeeringConnectionOptionsRead,
		Update: resourceAwsVpcPeeringConnectionOptionsUpdate,
		Delete: resourceAwsVpcPeeringConnectionOptionsDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"vpc_peering_connection_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"accepter":  vpcPeeringConnectionOptionsSchema(),
			"requester": vpcPeeringConnectionOptionsSchema(),
		},
	}
}

func resourceAwsVpcPeeringConnectionOptionsCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(d.Get("vpc_peering_connection_id").(string))
	return resourceAwsVpcPeeringConnectionOptionsUpdate(d, meta)
}

func resourceAwsVpcPeeringConnectionOptionsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	pcRaw, _, err := vpcPeeringConnectionRefreshState(conn, d.Id())()
	if err != nil {
		return fmt.Errorf("Error reading VPC Peering Connection: %s", err.Error())
	}

	if pcRaw == nil {
		log.Printf("[WARN] VPC Peering Connection (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	pc := pcRaw.(*ec2.VpcPeeringConnection)

	d.Set("vpc_peering_connection_id", pc.VpcPeeringConnectionId)

	if pc != nil && pc.AccepterVpcInfo != nil && pc.AccepterVpcInfo.PeeringOptions != nil {
		err := d.Set("accepter", flattenVpcPeeringConnectionOptions(pc.AccepterVpcInfo.PeeringOptions))
		if err != nil {
			return fmt.Errorf("Error setting VPC Peering Connection Options accepter information: %s", err.Error())
		}
	}

	if pc != nil && pc.RequesterVpcInfo != nil && pc.RequesterVpcInfo.PeeringOptions != nil {
		err := d.Set("requester", flattenVpcPeeringConnectionOptions(pc.RequesterVpcInfo.PeeringOptions))
		if err != nil {
			return fmt.Errorf("Error setting VPC Peering Connection Options requester information: %s", err.Error())
		}
	}

	return nil
}

func resourceAwsVpcPeeringConnectionOptionsUpdate(d *schema.ResourceData, meta interface{}) error {
	if err := resourceAwsVpcPeeringConnectionModifyOptions(d, meta); err != nil {
		return fmt.Errorf("Error modifying VPC Peering Connection Options: %s", err.Error())
	}

	return resourceAwsVpcPeeringConnectionOptionsRead(d, meta)
}

func resourceAwsVpcPeeringConnectionOptionsDelete(d *schema.ResourceData, meta interface{}) error {
	// Don't do anything with the underlying VPC peering connection.
	return nil
}
