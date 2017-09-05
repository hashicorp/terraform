package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsVpc() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsVpcRead,

		Schema: map[string]*schema.Schema{
			"cidr_block": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"dhcp_options_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"default": {
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

			"instance_tenancy": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv6_cidr_block": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv6_association_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"state": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"enable_dns_hostnames": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"enable_dns_support": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsVpcRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeVpcsInput{}

	if id := d.Get("id"); id != "" {
		req.VpcIds = []*string{aws.String(id.(string))}
	}

	// We specify "default" as boolean, but EC2 filters want
	// it to be serialized as a string. Note that setting it to
	// "false" here does not actually filter by it *not* being
	// the default, because Terraform can't distinguish between
	// "false" and "not set".
	isDefaultStr := ""
	if d.Get("default").(bool) {
		isDefaultStr = "true"
	}

	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"cidr":            d.Get("cidr_block").(string),
			"dhcp-options-id": d.Get("dhcp_options_id").(string),
			"isDefault":       isDefaultStr,
			"state":           d.Get("state").(string),
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

	log.Printf("[DEBUG] DescribeVpcs %s\n", req)
	resp, err := conn.DescribeVpcs(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.Vpcs) == 0 {
		return fmt.Errorf("no matching VPC found")
	}
	if len(resp.Vpcs) > 1 {
		return fmt.Errorf("multiple VPCs matched; use additional constraints to reduce matches to a single VPC")
	}

	vpc := resp.Vpcs[0]

	d.SetId(*vpc.VpcId)
	d.Set("id", vpc.VpcId)
	d.Set("cidr_block", vpc.CidrBlock)
	d.Set("dhcp_options_id", vpc.DhcpOptionsId)
	d.Set("instance_tenancy", vpc.InstanceTenancy)
	d.Set("default", vpc.IsDefault)
	d.Set("state", vpc.State)
	d.Set("tags", tagsToMap(vpc.Tags))

	if vpc.Ipv6CidrBlockAssociationSet != nil {
		d.Set("ipv6_association_id", vpc.Ipv6CidrBlockAssociationSet[0].AssociationId)
		d.Set("ipv6_cidr_block", vpc.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock)
	}

	attResp, err := awsVpcDescribeVpcAttribute("enableDnsSupport", *vpc.VpcId, conn)
	if err != nil {
		return err
	}
	d.Set("enable_dns_support", attResp.EnableDnsSupport.Value)

	attResp, err = awsVpcDescribeVpcAttribute("enableDnsHostnames", *vpc.VpcId, conn)
	if err != nil {
		return err
	}
	d.Set("enable_dns_hostnames", attResp.EnableDnsHostnames.Value)

	return nil
}
