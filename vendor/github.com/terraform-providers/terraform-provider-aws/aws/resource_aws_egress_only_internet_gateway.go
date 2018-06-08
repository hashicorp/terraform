package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/errwrap"
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

	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		igRaw, _, err := EIGWStateRefreshFunc(conn, d.Id())()
		if igRaw != nil {
			return nil
		}
		if err == nil {
			return resource.RetryableError(err)
		} else {
			return resource.NonRetryableError(err)
		}
	})

	if err != nil {
		return errwrap.Wrapf("{{err}}", err)
	}

	return resourceAwsEgressOnlyInternetGatewayRead(d, meta)
}

func EIGWStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeEgressOnlyInternetGateways(&ec2.DescribeEgressOnlyInternetGatewaysInput{
			EgressOnlyInternetGatewayIds: []*string{aws.String(id)},
		})
		if err != nil {
			ec2err, ok := err.(awserr.Error)
			if ok && ec2err.Code() == "InvalidEgressInternetGatewayID.NotFound" {
				resp = nil
			} else {
				log.Printf("[ERROR] Error on EIGWStateRefreshFunc: %s", err)
				return nil, "", err
			}
		}
		if len(resp.EgressOnlyInternetGateways) < 1 {
			resp = nil
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		ig := resp.EgressOnlyInternetGateways[0]
		return ig, "available", nil
	}
}

func resourceAwsEgressOnlyInternetGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	resp, err := conn.DescribeEgressOnlyInternetGateways(&ec2.DescribeEgressOnlyInternetGatewaysInput{
		EgressOnlyInternetGatewayIds: []*string{aws.String(d.Id())},
	})
	if err != nil {
		return fmt.Errorf("Error describing egress internet gateway: %s", err)
	}

	found := false
	for _, igw := range resp.EgressOnlyInternetGateways {
		if *igw.EgressOnlyInternetGatewayId == d.Id() {
			found = true
		}
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
