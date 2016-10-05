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
		// "resource aws_vpc_endpoint_route_table_association: All fields are ForceNew or Computed w/out Optional, Update is superfluous"
		// Update: resourceAwsVPCEndpointRouteTableAssociationUpdate,
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

	return resourceAwsVPCEndpointRouteTableAssociationRead(d, meta)
}

func resourceAwsVPCEndpointRouteTableAssociationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	endpointId := d.Get("vpc_endpoint_id").(string)
	rtId := d.Get("route_table_id").(string)

	err := findResourceVPCEndpointRouteTableAssociation(conn, endpointId, rtId)
	if _, notFound := err.(vpcEndpointNotFound); notFound {
		// The VPC endpoint containing this association no longer exists.
		d.SetId("")
		return nil
	}
	if _, notFound := err.(vpcEndpointRouteTableAssociationNotFound); notFound {
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
			log.Printf("[DEBUG] VPC endpoint is already gone")
		case "InvalidRouteTableId.NotFound":
			log.Printf("[DEBUG] Route table is already gone")
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
	if err, ok := err.(awserr.Error); ok && err.Code() == "InvalidVpcEndpointId.NotFound" {
		return nil, vpcEndpointNotFound{id, nil}
	}
	if err != nil {
		return nil, err
	}
	if output == nil {
		return nil, vpcEndpointNotFound{id, nil}
	}
	if len(output.VpcEndpoints) != 1 || output.VpcEndpoints[0] == nil {
		return nil, vpcEndpointNotFound{id, output.VpcEndpoints}
	}

	return output.VpcEndpoints[0], nil
}

func findResourceVPCEndpointRouteTableAssociation(conn *ec2.EC2, endpointId string, rtId string) error {
	vpce, err := findResourceVPCEndpoint(conn, endpointId)
	if err != nil {
		return err
	}
	for _, id := range vpce.RouteTableIds {
		if id != nil && *id == rtId {
			return nil
		}
	}

	return vpcEndpointRouteTableAssociationNotFound{endpointId, rtId}
}

func vpcEndpointIdRouteTableIdHash(endpointId, rtId string) string {
	return fmt.Sprintf("a-%s%d", endpointId, hashcode.String(rtId))
}

type vpcEndpointNotFound struct {
	id           string
	vpcEndpoints []*ec2.VpcEndpoint
}

func (err vpcEndpointNotFound) Error() string {
	if err.vpcEndpoints == nil {
		return fmt.Sprintf("No VPC endpoint with ID %q", err.id)
	}
	return fmt.Sprintf("Expected to find one VPC endpoint with ID %q, got: %#v",
		err.id, err.vpcEndpoints)
}

type vpcEndpointRouteTableAssociationNotFound struct {
	endpointId string
	rtId       string
}

func (err vpcEndpointRouteTableAssociationNotFound) Error() string {
	return fmt.Sprintf("Unable to find matching association for VPC endpoint (%s) and route table (%s)",
		err.endpointId,
		err.rtId)
}
