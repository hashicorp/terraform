package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDefaultVpcDhcpOptions() *schema.Resource {
	// reuse aws_vpc_dhcp_options schema, and methods for READ, UPDATE
	dvpc := resourceAwsVpcDhcpOptions()
	dvpc.Create = resourceAwsDefaultVpcDhcpOptionsCreate
	dvpc.Delete = resourceAwsDefaultVpcDhcpOptionsDelete

	// domain_name is a computed value for Default Default DHCP Options Sets
	dvpc.Schema["domain_name"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	// domain_name_servers is a computed value for Default Default DHCP Options Sets
	dvpc.Schema["domain_name_servers"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	// ntp_servers is a computed value for Default Default DHCP Options Sets
	dvpc.Schema["ntp_servers"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}

	return dvpc
}

func resourceAwsDefaultVpcDhcpOptionsCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	var domainName string
	awsRegion := meta.(*AWSClient).region
	if awsRegion == "us-east-1" {
		domainName = "ec2.internal"
	} else {
		domainName = awsRegion + ".compute.internal"
	}
	req := &ec2.DescribeDhcpOptionsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("key"),
				Values: aws.StringSlice([]string{"domain-name"}),
			},
			&ec2.Filter{
				Name:   aws.String("value"),
				Values: aws.StringSlice([]string{domainName}),
			},
			&ec2.Filter{
				Name:   aws.String("key"),
				Values: aws.StringSlice([]string{"domain-name-servers"}),
			},
			&ec2.Filter{
				Name:   aws.String("value"),
				Values: aws.StringSlice([]string{"AmazonProvidedDNS"}),
			},
		},
	}

	resp, err := conn.DescribeDhcpOptions(req)
	if err != nil {
		return err
	}

	if len(resp.DhcpOptions) != 1 || resp.DhcpOptions[0] == nil {
		return fmt.Errorf("Default DHCP Options Set not found")
	}

	d.SetId(aws.StringValue(resp.DhcpOptions[0].DhcpOptionsId))

	if err := resourceAwsVpcDhcpOptionsUpdate(d, meta); err != nil {
		return err
	}

	return resourceAwsVpcDhcpOptionsRead(d, meta)
}

func resourceAwsDefaultVpcDhcpOptionsDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[WARN] Cannot destroy Default DHCP Options Set. Terraform will remove this resource from the state file, however resources may remain.")
	return nil
}
