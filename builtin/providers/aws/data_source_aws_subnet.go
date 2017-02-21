package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsSubnet() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsSubnetRead,

		Schema: map[string]*schema.Schema{
			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"cidr_block": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"default_for_az": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"filter": ec2CustomFiltersSchema(),

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"state": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"tags": tagsSchemaComputed(),

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsSubnetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeSubnetsInput{}

	if id := d.Get("id"); id != "" {
		req.SubnetIds = []*string{aws.String(id.(string))}
	}

	// We specify default_for_az as boolean, but EC2 filters want
	// it to be serialized as a string. Note that setting it to
	// "false" here does not actually filter by it *not* being
	// the default, because Terraform can't distinguish between
	// "false" and "not set".
	defaultForAzStr := ""
	if d.Get("default_for_az").(bool) {
		defaultForAzStr = "true"
	}

	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"availabilityZone": d.Get("availability_zone").(string),
			"cidrBlock":        d.Get("cidr_block").(string),
			"defaultForAz":     defaultForAzStr,
			"state":            d.Get("state").(string),
			"vpc-id":           d.Get("vpc_id").(string),
		},
	)
	req.Filters = append(req.Filters, buildEC2TagFilterList(
		tagsFromMap(d.Get("tags").(map[string]interface{})),
	)...)
	req.Filters = append(req.Filters, buildEC2CustomFilterList(
		d.Get("filter").(*schema.Set),
	)...)
	if len(req.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		req.Filters = nil
	}

	log.Printf("[DEBUG] DescribeSubnets %s\n", req)
	resp, err := conn.DescribeSubnets(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.Subnets) == 0 {
		return fmt.Errorf("no matching subnet found")
	}
	if len(resp.Subnets) > 1 {
		return fmt.Errorf("multiple subnets matched; use additional constraints to reduce matches to a single subnet")
	}

	subnet := resp.Subnets[0]

	d.SetId(*subnet.SubnetId)
	d.Set("id", subnet.SubnetId)
	d.Set("vpc_id", subnet.VpcId)
	d.Set("availability_zone", subnet.AvailabilityZone)
	d.Set("cidr_block", subnet.CidrBlock)
	d.Set("default_for_az", subnet.DefaultForAz)
	d.Set("state", subnet.State)
	d.Set("tags", tagsToMap(subnet.Tags))

	return nil
}
