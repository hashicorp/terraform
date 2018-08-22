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
		Create: resourceAwsVpcEndpointRouteTableAssociationCreate,
		Read:   resourceAwsVpcEndpointRouteTableAssociationRead,
		Delete: resourceAwsVpcEndpointRouteTableAssociationDelete,
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

func resourceAwsVpcEndpointRouteTableAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	endpointId := d.Get("vpc_endpoint_id").(string)
	rtId := d.Get("route_table_id").(string)

	_, err := findResourceVpcEndpoint(conn, endpointId)
	if err != nil {
		return err
	}

	_, err = conn.ModifyVpcEndpoint(&ec2.ModifyVpcEndpointInput{
		VpcEndpointId:    aws.String(endpointId),
		AddRouteTableIds: aws.StringSlice([]string{rtId}),
	})
	if err != nil {
		return fmt.Errorf("Error creating VPC Endpoint/Route Table association: %s", err.Error())
	}

	d.SetId(vpcEndpointIdRouteTableIdHash(endpointId, rtId))

	return resourceAwsVpcEndpointRouteTableAssociationRead(d, meta)
}

func resourceAwsVpcEndpointRouteTableAssociationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	endpointId := d.Get("vpc_endpoint_id").(string)
	rtId := d.Get("route_table_id").(string)

	vpce, err := findResourceVpcEndpoint(conn, endpointId)
	if err != nil {
		if isAWSErr(err, "InvalidVpcEndpointId.NotFound", "") {
			log.Printf("[WARN] VPC Endpoint (%s) not found, removing VPC Endpoint/Route Table association (%s) from state", endpointId, d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	found := false
	for _, id := range vpce.RouteTableIds {
		if aws.StringValue(id) == rtId {
			found = true
			break
		}
	}
	if !found {
		log.Printf("[WARN] VPC Endpoint/Route Table association (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	return nil
}

func resourceAwsVpcEndpointRouteTableAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	endpointId := d.Get("vpc_endpoint_id").(string)
	rtId := d.Get("route_table_id").(string)

	_, err := conn.ModifyVpcEndpoint(&ec2.ModifyVpcEndpointInput{
		VpcEndpointId:       aws.String(endpointId),
		RemoveRouteTableIds: aws.StringSlice([]string{rtId}),
	})
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

	return nil
}

func findResourceVpcEndpoint(conn *ec2.EC2, id string) (*ec2.VpcEndpoint, error) {
	resp, err := conn.DescribeVpcEndpoints(&ec2.DescribeVpcEndpointsInput{
		VpcEndpointIds: aws.StringSlice([]string{id}),
	})
	if err != nil {
		return nil, err
	}

	if resp.VpcEndpoints == nil || len(resp.VpcEndpoints) == 0 {
		return nil, fmt.Errorf("No VPC Endpoints were found for %s", id)
	}

	return resp.VpcEndpoints[0], nil
}

func vpcEndpointIdRouteTableIdHash(endpointId, rtId string) string {
	return fmt.Sprintf("a-%s%d", endpointId, hashcode.String(rtId))
}
