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

func resourceAwsVpcEndpointRouteTableAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVPCEndpointRouteTableAssociationCreate,
		Read:   resourceAwsVPCEndpointRouteTableAssociationRead,
		Delete: resourceAwsVPCEndpointRouteTableAssociationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"vpc_endpoint_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"route_table_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsVPCEndpointRouteTableAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	endpointId := d.Get("vpc_endpoint_id").(string)
	rtId := d.Get("route_table_id").(string)

	_, err := findResourceVPCEndpoint(conn, endpointId)
	if err != nil {
		return err
	}

	log.Printf(
		"[INFO] Creating VPC Endpoint/Route Table association: %s => %s",
		endpointId, rtId)

	input := &ec2.ModifyVpcEndpointInput{
		VpcEndpointId:    aws.String(endpointId),
		AddRouteTableIds: aws.StringSlice([]string{rtId}),
	}

	_, err = conn.ModifyVpcEndpoint(input)
	if err != nil {
		return fmt.Errorf("Error creating VPC Endpoint/Route Table association: %s", err.Error())
	}
	id := vpcEndpointIdRouteTableIdHash(endpointId, rtId)
	log.Printf("[DEBUG] VPC Endpoint/Route Table association %q created.", id)

	d.SetId(id)

	return resourceAwsVPCEndpointRouteTableAssociationRead(d, meta)
}

func resourceAwsVPCEndpointRouteTableAssociationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	endpointId := d.Get("vpc_endpoint_id").(string)
	rtId := d.Get("route_table_id").(string)

	vpce, err := findResourceVPCEndpoint(conn, endpointId)
	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "InvalidVpcEndpointId.NotFound" {
			d.SetId("")
			return nil
		}

		return err
	}

	found := false
	for _, id := range vpce.RouteTableIds {
		if id != nil && *id == rtId {
			found = true
			break
		}
	}
	if !found {
		// The association no longer exists.
		d.SetId("")
		return nil
	}

	id := vpcEndpointIdRouteTableIdHash(endpointId, rtId)
	log.Printf("[DEBUG] Computed VPC Endpoint/Route Table ID %s", id)
	d.SetId(id)

	return nil
}

func resourceAwsVPCEndpointRouteTableAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	endpointId := d.Get("vpc_endpoint_id").(string)
	rtId := d.Get("route_table_id").(string)

	input := &ec2.ModifyVpcEndpointInput{
		VpcEndpointId:       aws.String(endpointId),
		RemoveRouteTableIds: aws.StringSlice([]string{rtId}),
	}

	_, err := conn.ModifyVpcEndpoint(input)
	if err != nil {
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return fmt.Errorf("Error deleting VPC Endpoint/Route Table association: %s", err.Error())
		}

		switch ec2err.Code() {
		case "InvalidVpcEndpointId.NotFound":
			fallthrough
		case "InvalidRouteTableId.NotFound":
			fallthrough
		case "InvalidParameter":
			log.Printf("[DEBUG] VPC Endpoint/Route Table association is already gone")
		default:
			return fmt.Errorf("Error deleting VPC Endpoint/Route Table association: %s", err.Error())
		}
	}

	log.Printf("[DEBUG] VPC Endpoint/Route Table association %q deleted", d.Id())
	d.SetId("")

	return nil
}

func findResourceVPCEndpoint(conn *ec2.EC2, id string) (*ec2.VpcEndpoint, error) {
	input := &ec2.DescribeVpcEndpointsInput{
		VpcEndpointIds: aws.StringSlice([]string{id}),
	}

	log.Printf("[DEBUG] Reading VPC Endpoint: %q", id)
	output, err := conn.DescribeVpcEndpoints(input)
	if err != nil {
		return nil, err
	}

	if output.VpcEndpoints == nil {
		return nil, fmt.Errorf("No VPC Endpoints were found for %q", id)
	}

	return output.VpcEndpoints[0], nil
}

func vpcEndpointIdRouteTableIdHash(endpointId, rtId string) string {
	return fmt.Sprintf("a-%s%d", endpointId, hashcode.String(rtId))
}
