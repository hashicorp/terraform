package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEgressOnlyInternetGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEgressOnlyInternetGatewayCreate,
		Read:   resourceAwsEgressOnlyInternetGatewayRead,
		Delete: resourceAwsEgressOnlyInternetGatewayDelete,

		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsEgressOnlyInternetGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	resp, err := conn.CreateEgressOnlyInternetGateway(&ec2.CreateEgressOnlyInternetGatewayInput{
		VpcId: aws.String(d.Get("vpc_id").(string)),
	})
	if err != nil {
		return fmt.Errorf("Error creating egress internet gateway: %s", err)
	}

	d.SetId(*resp.EgressOnlyInternetGateway.EgressOnlyInternetGatewayId)

	if err != nil {
		return fmt.Errorf("%s", err)
	}

	return resourceAwsEgressOnlyInternetGatewayRead(d, meta)
}

func resourceAwsEgressOnlyInternetGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	var found bool
	var req = &ec2.DescribeEgressOnlyInternetGatewaysInput{
		EgressOnlyInternetGatewayIds: []*string{aws.String(d.Id())},
	}

	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		resp, err := conn.DescribeEgressOnlyInternetGateways(req)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		if resp != nil && len(resp.EgressOnlyInternetGateways) > 0 {
			for _, igw := range resp.EgressOnlyInternetGateways {
				if aws.StringValue(igw.EgressOnlyInternetGatewayId) == d.Id() {
					found = true
					break
				}
			}
		}
		if d.IsNewResource() && !found {
			return resource.RetryableError(fmt.Errorf("Egress Only Internet Gateway (%s) not found.", d.Id()))
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error describing egress internet gateway: %s", err)
	}

	if !found {
		log.Printf("[Error] Cannot find Egress Only Internet Gateway: %q", d.Id())
		d.SetId("")
		return nil
	}

	return nil
}

func resourceAwsEgressOnlyInternetGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	_, err := conn.DeleteEgressOnlyInternetGateway(&ec2.DeleteEgressOnlyInternetGatewayInput{
		EgressOnlyInternetGatewayId: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("Error deleting egress internet gateway: %s", err)
	}

	return nil
}
