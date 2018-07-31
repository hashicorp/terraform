package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpnGatewayAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVpnGatewayAttachmentCreate,
		Read:   resourceAwsVpnGatewayAttachmentRead,
		Delete: resourceAwsVpnGatewayAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vpn_gateway_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsVpnGatewayAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	vpcId := d.Get("vpc_id").(string)
	vgwId := d.Get("vpn_gateway_id").(string)

	createOpts := &ec2.AttachVpnGatewayInput{
		VpcId:        aws.String(vpcId),
		VpnGatewayId: aws.String(vgwId),
	}
	log.Printf("[DEBUG] VPN Gateway attachment options: %#v", *createOpts)

	_, err := conn.AttachVpnGateway(createOpts)
	if err != nil {
		return fmt.Errorf("Error attaching VPN Gateway %q to VPC %q: %s",
			vgwId, vpcId, err)
	}

	d.SetId(vpnGatewayAttachmentId(vpcId, vgwId))
	log.Printf("[INFO] VPN Gateway %q attachment ID: %s", vgwId, d.Id())

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"detached", "attaching"},
		Target:     []string{"attached"},
		Refresh:    vpnGatewayAttachmentStateRefresh(conn, vpcId, vgwId),
		Timeout:    15 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for VPN Gateway %q to attach to VPC %q: %s",
			vgwId, vpcId, err)
	}
	log.Printf("[DEBUG] VPN Gateway %q attached to VPC %q.", vgwId, vpcId)

	return resourceAwsVpnGatewayAttachmentRead(d, meta)
}

func resourceAwsVpnGatewayAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	vgwId := d.Get("vpn_gateway_id").(string)

	resp, err := conn.DescribeVpnGateways(&ec2.DescribeVpnGatewaysInput{
		VpnGatewayIds: []*string{aws.String(vgwId)},
	})

	if err != nil {
		awsErr, ok := err.(awserr.Error)
		if ok && awsErr.Code() == "InvalidVpnGatewayID.NotFound" {
			log.Printf("[WARN] VPN Gateway %q not found.", vgwId)
			d.SetId("")
			return nil
		}
		return err
	}

	vgw := resp.VpnGateways[0]
	if *vgw.State == "deleted" {
		log.Printf("[INFO] VPN Gateway %q appears to have been deleted.", vgwId)
		d.SetId("")
		return nil
	}

	vga := vpnGatewayGetAttachment(vgw)
	if len(vgw.VpcAttachments) == 0 || *vga.State == "detached" {
		d.Set("vpc_id", "")
		return nil
	}

	d.Set("vpc_id", *vga.VpcId)
	return nil
}

func resourceAwsVpnGatewayAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	vpcId := d.Get("vpc_id").(string)
	vgwId := d.Get("vpn_gateway_id").(string)

	if vpcId == "" {
		log.Printf("[DEBUG] Not detaching VPN Gateway %q as no VPC ID is set.", vgwId)
		return nil
	}

	_, err := conn.DetachVpnGateway(&ec2.DetachVpnGatewayInput{
		VpcId:        aws.String(vpcId),
		VpnGatewayId: aws.String(vgwId),
	})

	if err != nil {
		awsErr, ok := err.(awserr.Error)
		if ok {
			switch awsErr.Code() {
			case "InvalidVpnGatewayID.NotFound":
				return nil
			case "InvalidVpnGatewayAttachment.NotFound":
				return nil
			}
		}

		return fmt.Errorf("Error detaching VPN Gateway %q from VPC %q: %s",
			vgwId, vpcId, err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"attached", "detaching"},
		Target:     []string{"detached"},
		Refresh:    vpnGatewayAttachmentStateRefresh(conn, vpcId, vgwId),
		Timeout:    15 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for VPN Gateway %q to detach from VPC %q: %s",
			vgwId, vpcId, err)
	}
	log.Printf("[DEBUG] VPN Gateway %q detached from VPC %q.", vgwId, vpcId)

	return nil
}

func vpnGatewayAttachmentStateRefresh(conn *ec2.EC2, vpcId, vgwId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeVpnGateways(&ec2.DescribeVpnGatewaysInput{
			Filters: []*ec2.Filter{
				&ec2.Filter{
					Name:   aws.String("attachment.vpc-id"),
					Values: []*string{aws.String(vpcId)},
				},
			},
			VpnGatewayIds: []*string{aws.String(vgwId)},
		})

		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok {
				switch awsErr.Code() {
				case "InvalidVpnGatewayID.NotFound":
					fallthrough
				case "InvalidVpnGatewayAttachment.NotFound":
					return nil, "", nil
				}
			}

			return nil, "", err
		}

		vgw := resp.VpnGateways[0]
		if len(vgw.VpcAttachments) == 0 {
			return vgw, "detached", nil
		}

		vga := vpnGatewayGetAttachment(vgw)

		log.Printf("[DEBUG] VPN Gateway %q attachment status: %s", vgwId, *vga.State)
		return vgw, *vga.State, nil
	}
}

func vpnGatewayAttachmentId(vpcId, vgwId string) string {
	return fmt.Sprintf("vpn-attachment-%x", hashcode.String(fmt.Sprintf("%s-%s", vpcId, vgwId)))
}
