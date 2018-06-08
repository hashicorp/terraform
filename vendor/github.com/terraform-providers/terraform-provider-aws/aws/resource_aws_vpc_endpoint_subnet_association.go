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

func resourceAwsVpcEndpointSubnetAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVpcEndpointSubnetAssociationCreate,
		Read:   resourceAwsVpcEndpointSubnetAssociationRead,
		Delete: resourceAwsVpcEndpointSubnetAssociationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"vpc_endpoint_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"subnet_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsVpcEndpointSubnetAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	endpointId := d.Get("vpc_endpoint_id").(string)
	snId := d.Get("subnet_id").(string)

	_, err := findResourceVpcEndpoint(conn, endpointId)
	if err != nil {
		return err
	}

	_, err = conn.ModifyVpcEndpoint(&ec2.ModifyVpcEndpointInput{
		VpcEndpointId: aws.String(endpointId),
		AddSubnetIds:  aws.StringSlice([]string{snId}),
	})
	if err != nil {
		return fmt.Errorf("Error creating Vpc Endpoint/Subnet association: %s", err.Error())
	}

	d.SetId(vpcEndpointIdSubnetIdHash(endpointId, snId))

	return resourceAwsVpcEndpointSubnetAssociationRead(d, meta)
}

func resourceAwsVpcEndpointSubnetAssociationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	endpointId := d.Get("vpc_endpoint_id").(string)
	snId := d.Get("subnet_id").(string)

	vpce, err := findResourceVpcEndpoint(conn, endpointId)
	if err != nil {
		if isAWSErr(err, "InvalidVpcEndpointId.NotFound", "") {
			log.Printf("[WARN] Vpc Endpoint (%s) not found, removing Vpc Endpoint/Subnet association (%s) from state", endpointId, d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	found := false
	for _, id := range vpce.SubnetIds {
		if aws.StringValue(id) == snId {
			found = true
			break
		}
	}
	if !found {
		log.Printf("[WARN] Vpc Endpoint/Subnet association (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	return nil
}

func resourceAwsVpcEndpointSubnetAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	endpointId := d.Get("vpc_endpoint_id").(string)
	snId := d.Get("subnet_id").(string)

	_, err := conn.ModifyVpcEndpoint(&ec2.ModifyVpcEndpointInput{
		VpcEndpointId:   aws.String(endpointId),
		RemoveSubnetIds: aws.StringSlice([]string{snId}),
	})
	if err != nil {
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return fmt.Errorf("Error deleting Vpc Endpoint/Subnet association: %s", err.Error())
		}

		switch ec2err.Code() {
		case "InvalidVpcEndpointId.NotFound":
			fallthrough
		case "InvalidRouteTableId.NotFound":
			fallthrough
		case "InvalidParameter":
			log.Printf("[DEBUG] Vpc Endpoint/Subnet association is already gone")
		default:
			return fmt.Errorf("Error deleting Vpc Endpoint/Subnet association: %s", err.Error())
		}
	}

	return nil
}

func vpcEndpointIdSubnetIdHash(endpointId, snId string) string {
	return fmt.Sprintf("a-%s%d", endpointId, hashcode.String(snId))
}
