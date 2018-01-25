package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
)

func dataSourceAwsVpcEndpoint() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsVpcEndpointRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"service_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"policy": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"route_table_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"prefix_list_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsVpcEndpointRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeVpcEndpointsInput{}

	if id, ok := d.GetOk("id"); ok {
		req.VpcEndpointIds = aws.StringSlice([]string{id.(string)})
	}

	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"vpc-endpoint-state": d.Get("state").(string),
			"vpc-id":             d.Get("vpc_id").(string),
			"service-name":       d.Get("service_name").(string),
		},
	)
	if len(req.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		req.Filters = nil
	}

	log.Printf("[DEBUG] Reading VPC Endpoint: %s", req)
	resp, err := conn.DescribeVpcEndpoints(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.VpcEndpoints) == 0 {
		return fmt.Errorf("no matching VPC endpoint found")
	}
	if len(resp.VpcEndpoints) > 1 {
		return fmt.Errorf("multiple VPC endpoints matched; use additional constraints to reduce matches to a single VPC endpoint")
	}

	vpce := resp.VpcEndpoints[0]
	policy, err := structure.NormalizeJsonString(*vpce.PolicyDocument)
	if err != nil {
		return errwrap.Wrapf("policy contains an invalid JSON: {{err}}", err)
	}

	prefixListServiceName := *vpce.ServiceName
	prefixListInput := &ec2.DescribePrefixListsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("prefix-list-name"), Values: []*string{aws.String(prefixListServiceName)}},
		},
	}
	log.Printf("[DEBUG] Reading VPC Endpoint prefix list: %s", prefixListServiceName)
	prefixListsOutput, err := conn.DescribePrefixLists(prefixListInput)

	if err != nil {
		_, ok := err.(awserr.Error)
		if !ok {
			return fmt.Errorf("Error reading VPC Endpoint prefix list: %s", err.Error())
		}
	}

	if len(prefixListsOutput.PrefixLists) != 1 {
		return fmt.Errorf("There are multiple prefix lists associated with the service name '%s'. Unexpected", prefixListServiceName)
	}

	d.SetId(aws.StringValue(vpce.VpcEndpointId))
	d.Set("state", vpce.State)
	d.Set("vpc_id", vpce.VpcId)
	d.Set("service_name", vpce.ServiceName)
	d.Set("policy", policy)

	pl := prefixListsOutput.PrefixLists[0]
	d.Set("prefix_list_id", pl.PrefixListId)

	if err := d.Set("route_table_ids", aws.StringValueSlice(vpce.RouteTableIds)); err != nil {
		return err
	}

	return nil
}
