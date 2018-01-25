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
			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"cidr_block": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"ipv6_cidr_block": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"default_for_az": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"filter": ec2CustomFiltersSchema(),

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

			"tags": tagsSchemaComputed(),

			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"assign_ipv6_address_on_creation": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"map_public_ip_on_launch": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"ipv6_cidr_block_association_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsSubnetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeSubnetsInput{}

	if id, ok := d.GetOk("id"); ok {
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

	filters := map[string]string{
		"availabilityZone": d.Get("availability_zone").(string),
		"defaultForAz":     defaultForAzStr,
		"state":            d.Get("state").(string),
		"vpc-id":           d.Get("vpc_id").(string),
	}

	if v, ok := d.GetOk("cidr_block"); ok {
		filters["cidrBlock"] = v.(string)
	}

	if v, ok := d.GetOk("ipv6_cidr_block"); ok {
		filters["ipv6-cidr-block-association.ipv6-cidr-block"] = v.(string)
	}

	req.Filters = buildEC2AttributeFilterList(filters)
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

	log.Printf("[DEBUG] Reading Subnet: %s", req)
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
	d.Set("vpc_id", subnet.VpcId)
	d.Set("availability_zone", subnet.AvailabilityZone)
	d.Set("cidr_block", subnet.CidrBlock)
	d.Set("default_for_az", subnet.DefaultForAz)
	d.Set("state", subnet.State)
	d.Set("tags", tagsToMap(subnet.Tags))
	d.Set("assign_ipv6_address_on_creation", subnet.AssignIpv6AddressOnCreation)
	d.Set("map_public_ip_on_launch", subnet.MapPublicIpOnLaunch)

	for _, a := range subnet.Ipv6CidrBlockAssociationSet {
		if *a.Ipv6CidrBlockState.State == "associated" { //we can only ever have 1 IPv6 block associated at once
			d.Set("ipv6_cidr_block_association_id", a.AssociationId)
			d.Set("ipv6_cidr_block", a.Ipv6CidrBlock)
		}
	}

	return nil
}
