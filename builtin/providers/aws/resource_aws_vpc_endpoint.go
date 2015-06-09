package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpcEndpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVPCEndpointCreate,
		Read:   resourceAwsVPCEndpointRead,
		Update: resourceAwsVPCEndpointUpdate,
		Delete: resourceAwsVPCEndpointDelete,
		Schema: map[string]*schema.Schema{
			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"route_table_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
		},
	}
}

func resourceAwsVPCEndpointCreate(d *schema.ResourceData, meta interface{}) error {
	svc := meta.(*AWSClient).ec2conn
	input := &ec2.CreateVPCEndpointInput{
		VPCID:         aws.String(d.Get("vpc_id").(string)),
		RouteTableIDs: expandStringList(d.Get("route_table_ids").(*schema.Set).List()),
		ServiceName:   aws.String(d.Get("service_name").(string)),
	}

	if v := d.Get("policy"); v != nil {
		input.PolicyDocument = aws.String(v.(string))
	}

	output, err := svc.CreateVPCEndpoint(input)
	if err != nil {
		return fmt.Errorf("Error creating vpc endpoint: %s", err)
	}

	d.SetId(*output.VPCEndpoint.VPCEndpointID)

	if input.PolicyDocument == nil {
		d.Set("policy", output.VPCEndpoint.PolicyDocument)
	}

	return nil
}

func resourceAwsVPCEndpointRead(d *schema.ResourceData, meta interface{}) error {
	svc := meta.(*AWSClient).ec2conn
	input := &ec2.DescribeVPCEndpointsInput{
		VPCEndpointIDs: []*string{aws.String(d.Id())},
	}

	output, err := svc.DescribeVPCEndpoints(input)
	if err != nil {
		return fmt.Errorf("Error reading vpc endpoint: %s", err)
	}

	if len(output.VPCEndpoints) != 1 {
		return fmt.Errorf("Error reading vpc endpoint: %s", err)
	}

	vpce := output.VPCEndpoints[0]

	d.Set("state", vpce.State)
	d.Set("vpc_id", vpce.VPCID)
	d.Set("policy", vpce.PolicyDocument)
	d.Set("service_name", vpce.ServiceName)
	d.Set("route_table_ids", vpce.RouteTableIDs)
	return nil
}

func resourceAwsVPCEndpointUpdate(d *schema.ResourceData, meta interface{}) error {
	svc := meta.(*AWSClient).ec2conn
	input := &ec2.ModifyVPCEndpointInput{
		VPCEndpointID: aws.String(d.Id()),
	}

	if d.HasChange("route_table_ids") {
		o, n := d.GetChange("route_table_ids")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		add := expandStringList(os.Difference(ns).List())
		if len(add) > 0 {
			input.AddRouteTableIDs = add
		}

		remove := expandStringList(ns.Difference(os).List())
		if len(remove) > 0 {
			input.RemoveRouteTableIDs = remove
		}
	}

	if d.HasChange("policy") {
		input.PolicyDocument = aws.String(d.Get("policy").(string))
		input.ResetPolicy = aws.Boolean(true)
	} else {
		input.ResetPolicy = aws.Boolean(false)
	}

	_, err := svc.ModifyVPCEndpoint(input)
	if err != nil {
		return fmt.Errorf("Error updating vpc endpoint: %s", err)
	}

	return nil
}

func resourceAwsVPCEndpointDelete(d *schema.ResourceData, meta interface{}) error {
	svc := meta.(*AWSClient).ec2conn
	input := &ec2.DeleteVPCEndpointsInput{
		VPCEndpointIDs: []*string{aws.String(d.Id())},
	}

	_, err := svc.DeleteVPCEndpoints(input)
	if err != nil {
		return fmt.Errorf("Error deleting vpc endpoint: %s", err)
	}

	return nil
}
