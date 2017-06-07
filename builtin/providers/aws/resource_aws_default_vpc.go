package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDefaultVpc() *schema.Resource {
	// reuse aws_vpc schema, and methods for READ, UPDATE
	dvpc := resourceAwsVpc()
	dvpc.Create = resourceAwsDefaultVpcCreate
	dvpc.Delete = resourceAwsDefaultVpcDelete

	// cidr_block is a computed value for Default VPCs
	dvpc.Schema["cidr_block"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	// instance_tenancy is a computed value for Default VPCs
	dvpc.Schema["instance_tenancy"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	// assign_generated_ipv6_cidr_block is a computed value for Default VPCs
	dvpc.Schema["assign_generated_ipv6_cidr_block"] = &schema.Schema{
		Type:     schema.TypeBool,
		Computed: true,
	}

	return dvpc
}

func resourceAwsDefaultVpcCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	req := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("isDefault"),
				Values: aws.StringSlice([]string{"true"}),
			},
		},
	}

	resp, err := conn.DescribeVpcs(req)
	if err != nil {
		return err
	}

	if resp.Vpcs == nil || len(resp.Vpcs) == 0 {
		return fmt.Errorf("No default VPC found in this region.")
	}

	d.SetId(aws.StringValue(resp.Vpcs[0].VpcId))

	return resourceAwsVpcUpdate(d, meta)
}

func resourceAwsDefaultVpcDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[WARN] Cannot destroy Default VPC. Terraform will remove this resource from the state file, however resources may remain.")
	d.SetId("")
	return nil
}
