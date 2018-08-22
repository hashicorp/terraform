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

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
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

	// See https://github.com/terraform-providers/terraform-provider-aws/issues/3382.
	// Prevent concurrent subnet association requests and delay between requests.
	mk := "vpc_endpoint_subnet_association_" + endpointId
	awsMutexKV.Lock(mk)
	defer awsMutexKV.Unlock(mk)

	c := &resource.StateChangeConf{
		Delay:   1 * time.Minute,
		Timeout: 3 * time.Minute,
		Target:  []string{"ok"},
		Refresh: func() (interface{}, string, error) {
			res, err := conn.ModifyVpcEndpoint(&ec2.ModifyVpcEndpointInput{
				VpcEndpointId: aws.String(endpointId),
				AddSubnetIds:  aws.StringSlice([]string{snId}),
			})
			return res, "ok", err
		},
	}
	_, err = c.WaitForState()
	if err != nil {
		return fmt.Errorf("Error creating Vpc Endpoint/Subnet association: %s", err)
	}

	d.SetId(vpcEndpointSubnetAssociationId(endpointId, snId))

	if err := vpcEndpointWaitUntilAvailable(conn, endpointId, d.Timeout(schema.TimeoutCreate)); err != nil {
		return err
	}

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
			return fmt.Errorf("Error deleting Vpc Endpoint/Subnet association: %s", err)
		}

		switch ec2err.Code() {
		case "InvalidVpcEndpointId.NotFound":
			fallthrough
		case "InvalidParameter":
			log.Printf("[DEBUG] Vpc Endpoint/Subnet association is already gone")
		default:
			return fmt.Errorf("Error deleting Vpc Endpoint/Subnet association: %s", err)
		}
	}

	if err := vpcEndpointWaitUntilAvailable(conn, endpointId, d.Timeout(schema.TimeoutDelete)); err != nil {
		return err
	}

	return nil
}

func vpcEndpointSubnetAssociationId(endpointId, snId string) string {
	return fmt.Sprintf("a-%s%d", endpointId, hashcode.String(snId))
}
