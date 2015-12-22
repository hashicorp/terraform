package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: normalizeJson,
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
	conn := meta.(*AWSClient).ec2conn
	input := &ec2.CreateVpcEndpointInput{
		VpcId:         aws.String(d.Get("vpc_id").(string)),
		RouteTableIds: expandStringList(d.Get("route_table_ids").(*schema.Set).List()),
		ServiceName:   aws.String(d.Get("service_name").(string)),
	}

	if v, ok := d.GetOk("policy"); ok {
		policy := normalizeJson(v)
		input.PolicyDocument = aws.String(policy)
	}

	log.Printf("[DEBUG] Creating VPC Endpoint: %#v", input)
	output, err := conn.CreateVpcEndpoint(input)
	if err != nil {
		return fmt.Errorf("Error creating VPC Endpoint: %s", err)
	}
	log.Printf("[DEBUG] VPC Endpoint %q created.", *output.VpcEndpoint.VpcEndpointId)

	d.SetId(*output.VpcEndpoint.VpcEndpointId)

	return resourceAwsVPCEndpointRead(d, meta)
}

func resourceAwsVPCEndpointRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	input := &ec2.DescribeVpcEndpointsInput{
		VpcEndpointIds: []*string{aws.String(d.Id())},
	}

	log.Printf("[DEBUG] Reading VPC Endpoint: %q", d.Id())
	output, err := conn.DescribeVpcEndpoints(input)

	if err != nil {
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return fmt.Errorf("Error reading VPC Endpoint: %s", err.Error())
		}

		if ec2err.Code() == "InvalidVpcEndpointId.NotFound" {
			return nil
		}

		return fmt.Errorf("Error reading VPC Endpoint: %s", err.Error())
	}

	if len(output.VpcEndpoints) != 1 {
		return fmt.Errorf("There's no unique VPC Endpoint, but %d endpoints: %#v",
			len(output.VpcEndpoints), output.VpcEndpoints)
	}

	vpce := output.VpcEndpoints[0]

	d.Set("vpc_id", vpce.VpcId)
	d.Set("policy", normalizeJson(*vpce.PolicyDocument))
	d.Set("service_name", vpce.ServiceName)
	if err := d.Set("route_table_ids", aws.StringValueSlice(vpce.RouteTableIds)); err != nil {
		return err
	}

	return nil
}

func resourceAwsVPCEndpointUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	input := &ec2.ModifyVpcEndpointInput{
		VpcEndpointId: aws.String(d.Id()),
	}

	if d.HasChange("route_table_ids") {
		o, n := d.GetChange("route_table_ids")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		add := expandStringList(ns.Difference(os).List())
		if len(add) > 0 {
			input.AddRouteTableIds = add
		}

		remove := expandStringList(os.Difference(ns).List())
		if len(remove) > 0 {
			input.RemoveRouteTableIds = remove
		}
	}

	if d.HasChange("policy") {
		policy := normalizeJson(d.Get("policy"))
		input.PolicyDocument = aws.String(policy)
	}

	log.Printf("[DEBUG] Updating VPC Endpoint: %#v", input)
	_, err := conn.ModifyVpcEndpoint(input)
	if err != nil {
		return fmt.Errorf("Error updating VPC Endpoint: %s", err)
	}
	log.Printf("[DEBUG] VPC Endpoint %q updated", input.VpcEndpointId)

	return resourceAwsVPCEndpointRead(d, meta)
}

func resourceAwsVPCEndpointDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	input := &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: []*string{aws.String(d.Id())},
	}

	log.Printf("[DEBUG] Deleting VPC Endpoint: %#v", input)
	_, err := conn.DeleteVpcEndpoints(input)

	if err != nil {
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return fmt.Errorf("Error deleting VPC Endpoint: %s", err.Error())
		}

		if ec2err.Code() == "InvalidVpcEndpointId.NotFound" {
			log.Printf("[DEBUG] VPC Endpoint %q is already gone", d.Id())
		} else {
			return fmt.Errorf("Error deleting VPC Endpoint: %s", err.Error())
		}
	}

	log.Printf("[DEBUG] VPC Endpoint %q deleted", d.Id())
	d.SetId("")

	return nil
}
